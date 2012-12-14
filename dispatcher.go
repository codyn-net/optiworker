package main

// #include <sys/time.h>
// #include <sys/resource.h>
import "C"

import (
	"bytes"
	"fmt"
	"io"
	"ponyo.epfl.ch/go/get/optimization/go/optimization/messages/task.pb"
	"os"
	"os/exec"
	"ponyo.epfl.ch/go/get/optimization/go/optimization"
	"strings"
	"syscall"
	"time"
)

type Dispatcher struct {
	Id      uint
	Master *Master
	Task   *task.Task

	cmd    *exec.Cmd
	stderr *bytes.Buffer
	stdin   io.WriteCloser

	Running              bool
	AuthenticationNeeded bool
	HasResponse          bool

	OnResponse *optimization.Signal
}

func NewDispatcher(id uint, master *Master, t *task.Task) *Dispatcher {
	return &Dispatcher{
		Id:         id,
		Master:     master,
		Task:       t,
		Running:    false,
		OnResponse: optimization.NewSyncSignal(func(*task.Response) {}),
	}
}

func (x *Dispatcher) SendTokenResponse(response string) error {
	if !x.AuthenticationNeeded {
		return nil
	}

	x.stdin.Write([]byte(response))
	return nil
}

func (x *Dispatcher) Fail(tp task.Response_Failure_Type, message string) {
	r := x.Master.MakeResponse(x.Task, task.Response_Failed)

	if x.stderr != nil && x.stderr.Len() != 0 {
		message += fmt.Sprintf("\nDispatcher: %v", x.stderr.String())
	}

	x.stderr = nil

	r.Failure.Type = &tp
	r.Failure.Message = &message

	x.OnResponse.Emit(r)
}

func (x *Dispatcher) splitEnv(environ []string, envmap map[string]string) {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)

		if len(parts) == 1 {
			envmap[parts[0]] = ""
		} else {
			envmap[parts[0]] = parts[1]
		}
	}
}

func (x *Dispatcher) joinEnv(envmap map[string]string) []string {
	ret := make([]string, 0)

	for k, v := range envmap {
		ret = append(ret, fmt.Sprintf("%v=%v", k, v))
	}

	return ret
}

func (x *Dispatcher) Run() {
	if x.Running {
		return
	}

	var err error

	// Find dispatcher to execute
	dispatcher := FindDispatcher(x.Task.GetDispatcher())

	if dispatcher == "" {
		x.Fail(task.Response_Failure_DispatcherNotFound,
			fmt.Sprintf("Unable to find dispatcher '%v'", x.Task.GetDispatcher()))

		return
	}

	// Prepare environment
	envmap := make(map[string]string)
	x.splitEnv(os.Environ(), envmap)

	for _, setting := range x.Task.Settings {
		if setting.GetKey() != "environment" {
			continue
		}

		parts := strings.Split(setting.GetValue(), ",")
		x.splitEnv(parts, envmap)
	}

	envmap["OPTIWORKER_PROCESS_NUMBER"] = fmt.Sprintf("%v", x.Id)

	if TheConfig.UseAuthentication {
		optirooter, err := exec.LookPath("optirooter")

		if err != nil {
			x.Fail(task.Response_Failure_DispatcherNotFound,
				fmt.Sprintf("Unable to find optirooter: %v", err))

			return
		}

		if ldpath, ok := envmap["LD_LIBRARY_PATH"]; ok {
			envmap["SAVE_LD_LIBRARY_PATH"] = ldpath
		}

		// Launch optirooter with dispatcher as the first argument
		x.cmd = exec.Command(optirooter, dispatcher)
		x.cmd.Env = x.joinEnv(envmap)
		x.cmd.Dir, _ = os.Getwd()

		x.AuthenticationNeeded = true
	} else {
		// Launch dispatcher
		x.cmd = exec.Command(dispatcher)
		x.cmd.Env = x.joinEnv(envmap)
		x.cmd.Dir, _ = os.Getwd()

		x.AuthenticationNeeded = false
	}

	x.HasResponse = false

	defer func() {
		if err != nil {
			x.cmd = nil
			x.AuthenticationNeeded = false

			x.Fail(task.Response_Failure_DispatcherNotFound,
			       fmt.Sprintf("Failed to run dispatcher: %v", err))
		}
	}()

	// Start reading from stdout and stderr
	stdout, err := x.cmd.StdoutPipe()

	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			stdout.Close()
		}
	}()

	go x.readStdout(stdout)

	stderr, err := x.cmd.StderrPipe()

	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			stderr.Close()
		}
	}()

	x.stderr = new(bytes.Buffer)
	go x.readStderr(stderr)

	stdin, err := x.cmd.StdinPipe()

	if err != nil {
		return
	}

	x.stdin = stdin

	defer func() {
		if err != nil {
			stdin.Close()
		}
	}()

	err = x.cmd.Start()

	if err != nil {
		return
	}

	x.Running = true

	// Set process group to same as the worker
	syscall.Setpgid(x.cmd.Process.Pid, syscall.Getpgrp())

	// Set priority of the dispatcher according to the setting
	if TheConfig.DispatcherPriority > 0 {
		C.setpriority(C.PRIO_PROCESS,
			C.id_t(x.cmd.Process.Pid),
			C.int(TheConfig.DispatcherPriority))
	}

	if !x.AuthenticationNeeded {
		x.WriteTask()
	}
}

func (x *Dispatcher) WriteTask() {
	data, err := optimization.EncodeCommunication(x.Task)

	if err != nil {
		x.Fail(task.Response_Failure_WrongRequest,
			fmt.Sprintf("Failed to encode task: %v", err))

		return
	}

	stdin := x.stdin

	go func() {
		stdin.Write(data)
		stdin.Close()
	}()
}

func (x *Dispatcher) readAll(reader io.Reader, buf []byte) error {
	pos := 0

	for pos < len(buf) {
		n, err := reader.Read(buf[pos:])
		pos += n

		if pos < len(buf) && err != nil {
			return err
		}
	}

	return nil
}

func (x *Dispatcher) readStderr(reader io.Reader) {
	cmd := x.cmd

	data := make([]byte, 512)

	for {
		n, err := reader.Read(data)

		if cmd != x.cmd {
			break
		}

		datacp := make([]byte, n)
		copy(datacp, data[:n])

		optimization.Events <- func() {
			if cmd == x.cmd && x.stderr != nil {
				x.stderr.Write(datacp)
			}
		}

		if err != nil {
			break
		}
	}
}

func (x *Dispatcher) readStdout(reader io.Reader) {
	cmd := x.cmd

	if x.AuthenticationNeeded {
		// Read exactly 33 bytes
		challenge := make([]byte, 33)
		err := x.readAll(reader, challenge)

		// Make sure we are still reading for the current command. Could
		// have been cancelled in the mean time
		if x.cmd != cmd {
			return
		}

		if err != nil {
			// Big fail
			optimization.Events <- func() {
				x.Fail(task.Response_Failure_WrongRequest,
					fmt.Sprintf("Unable to read challenge: %v", err))
			}

			return
		}

		// Send challenge to master
		x.Master.Challenge(x.Task, string(challenge[0:len(challenge)-1]))
	}

	optimization.ReadCommunication(reader, new(task.Response), func(msg interface{}, err error) bool {
		// Make sure we are still reading for the current command
		if x.cmd != cmd {
			return false
		}

		if err == nil {
			x.HasResponse = true

			r := msg.(*task.Response)

			r.Id = x.Task.Id
			r.Uniqueid = x.Task.Uniqueid

			x.OnResponse.EmitAsync(msg)
		} else if !x.HasResponse {
			optimization.Events <- func() {
				x.Fail(task.Response_Failure_NoResponse,
					fmt.Sprintf("Unable to read response: %v", err))
			}
		}

		return false
	})
}

func (x *Dispatcher) Cancel() {
	if !x.Running {
		return
	}

	// Kill dispatcher
	x.cmd.Process.Signal(syscall.SIGTERM)

	cmd := x.cmd
	x.cmd = nil

	go func() {
		// Set a timeout to kill the process real hard
		optimization.Events.Timeout(3*time.Second, func() {
			// Only kill the process if it hasn't been terminated
			if cmd != nil {
				cmd.Process.Kill()
			}
		})

		// Wait for cmd to terminate
		cmd.Wait()
		cmd = nil
	}()

	x.AuthenticationNeeded = false
	x.Running = false
	x.HasResponse = false
	x.stderr = nil
}
