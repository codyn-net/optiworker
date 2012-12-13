package main

import (
	"container/list"
	"fmt"
	"net"
	task "optimization/messages/task.pb"
	"ponyo.epfl.ch/go/get/optimization/go/optimization"
	"ponyo.epfl.ch/go/get/optimization/go/optimization/log"
	optinet "ponyo.epfl.ch/go/get/optimization/go/optimization/net"
	"strconv"
)

type DispatcherQueue struct {
	list.List
}

func (x *DispatcherQueue) Push(t *Dispatcher) {
	x.PushBack(t)
}

func (x *DispatcherQueue) Pop() *Dispatcher {
	if !x.Empty() {
		return x.Remove(x.Front()).(*Dispatcher)
	}

	return nil
}

func (x *DispatcherQueue) Peek() *Dispatcher {
	front := x.Front()

	if front == nil {
		return nil
	}

	return front.Value.(*Dispatcher)
}

func (x *DispatcherQueue) RemoveWithMaster(master *Master) {
	var elem *list.Element

	elem = nil

	for {
		if elem == nil {
			elem = x.Front()

			if elem == nil {
				break
			}
		}

		if elem.Value.(*Master) == master {
			prev := elem.Prev()

			dispatcher := x.Remove(elem).(*Dispatcher)
			dispatcher.Cancel()

			elem = prev
		}
	}
}

func (x *DispatcherQueue) Empty() bool {
	return x.Front() == nil
}

type Dispatch struct {
	conn          net.Listener
	listenAddress string

	masters     []*Master
	dispatchers *DispatcherQueue

	Id uint
}

func (x *Dispatch) accept() {
	for {
		remote, err := x.conn.Accept()

		if err != nil {
			return
		}

		master := &Master{
			Client: optimization.NewClientConnection(remote, new(task.Communication)),
		}

		log.W("Master connected from %v", remote.RemoteAddr().String())

		master.OnMessage.Connect(func(msg *task.Communication) {
			x.Handle(master, msg)
		})

		optimization.Events <- func() {
			x.masters = append(x.masters, master)
		}

		master.OnState.Connect(func() {
			if master.State == optimization.Disconnected {
				for i, v := range x.masters {
					if v == master {
						x.masters = append(x.masters[:i], x.masters[i+1:]...)
					}
				}

				x.dispatchers.RemoveWithMaster(master)
			}
		})
	}
}

func NewDispatch(address string, id uint) (*Dispatch, error) {
	ret := new(Dispatch)

	ret.dispatchers = new(DispatcherQueue)
	ret.Id = id

	naddr := optinet.ParseAddress(address)
	err := naddr.Resolve()

	if err != nil {
		return nil, err
	}

	if id != 0 {
		i, _ := strconv.ParseUint(naddr.Port, 10, 32)

		if i != 0 {
			naddr.Port = fmt.Sprintf("%v", uint(i)+id)
		}
	}

	ret.conn, err = naddr.Listen()

	if err != nil {
		return nil, err
	}

	ret.listenAddress = naddr.String()

	log.W("Started dispatch server on `%s'", ret.ListenAddress())

	go ret.accept()
	return ret, nil
}

func (x *Dispatch) ListenAddress() string {
	return x.listenAddress
}

func (x *Dispatch) checkCurrentDispatcher(master *Master, id uint32) bool {
	if x.dispatchers.Empty() {
		return false
	}

	dispatcher := x.dispatchers.Peek()

	if !dispatcher.Running {
		return false
	}

	if dispatcher.Master != master {
		return false
	}

	if dispatcher.Task.GetId() != id {
		return false
	}

	return true
}

func (x *Dispatch) Dispatch() {
	dispatcher := x.dispatchers.Peek()

	if dispatcher == nil || dispatcher.Running {
		return
	}

	log.W("Dispatching task: %v, %v", dispatcher.Task.GetId(), dispatcher.Task.GetUniqueid())

	dispatcher.OnResponse.Connect(func(r *task.Response) {
		// Relay response to master
		log.W("Task finished: %v, %v: (%v, %v): %v",
		      dispatcher.Task.GetId(),
		      dispatcher.Task.GetUniqueid(),
		      r.GetId(),
		      r.GetUniqueid(),
		      r.GetStatus())

		if r.GetStatus() == task.Response_Failed {
			log.W("Failed because: %s", r.GetFailure())
		}

		dispatcher.Master.Respond(r)
		x.Finished(dispatcher)
	})

	dispatcher.Run()
}

func (x *Dispatch) handleTask(master *Master, msg *task.Task) {
	log.W("Received task: %v, %v", msg.GetId(), msg.GetUniqueid())

	x.dispatchers.Push(NewDispatcher(x.Id, master, msg))
	x.Dispatch()
}

func (x *Dispatch) handleToken(master *Master, msg *task.Token) {
	if !x.checkCurrentDispatcher(master, msg.GetId()) {
		return
	}

	dispatcher := x.dispatchers.Peek()
	dispatcher.SendTokenResponse(msg.GetResponse())

	dispatcher.WriteTask()
}

func (x *Dispatch) Finished(dispatcher *Dispatcher) {
	dispatcher.Cancel()

	if x.dispatchers.Peek() == dispatcher {
		x.dispatchers.Pop()
		x.Dispatch()
	}
}

func (x *Dispatch) handleCancel(master *Master, msg *task.Cancel) {
	if !x.checkCurrentDispatcher(master, msg.GetId()) {
		return
	}

	x.Finished(x.dispatchers.Peek())
}

func (x *Dispatch) Handle(master *Master, msg *task.Communication) {
	if master.State == optimization.Disconnected {
		return
	}

	switch msg.GetType() {
	case task.Communication_CommunicationTask:
		x.handleTask(master, msg.GetTask())
	case task.Communication_CommunicationToken:
		x.handleToken(master, msg.GetToken())
	case task.Communication_CommunicationCancel:
		x.handleCancel(master, msg.GetCancel())
	}
}

func (x *Dispatch) Close() {
	dispatcher := x.dispatchers.Peek()

	if dispatcher != nil {
		dispatcher.Cancel()
	}
}
