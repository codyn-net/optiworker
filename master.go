package main

import (
	task "optimization/messages/task.pb"
	"ponyo.epfl.ch/go/get/optimization/go/optimization"
)

type Master struct {
	*optimization.Client
}

func (x *Master) MakeResponse(t *task.Task, status task.Response_Status) *task.Response {
	r := new(task.Response)

	r.Id = t.Id
	r.Uniqueid = t.Uniqueid
	r.Status = &status

	if status == task.Response_Failed {
		r.Failure = new(task.Response_Failure)
		ftp := task.Response_Failure_NoResponse

		r.Failure.Type = &ftp
	}

	return r
}

func (x *Master) wrapResponse(r *task.Response) *task.Communication {
	c := new(task.Communication)

	tp := task.Communication_CommunicationResponse
	c.Type = &tp

	c.Response = r

	return c
}

func (x *Master) MakeCommunicationResponse(t *task.Task, status task.Response_Status) *task.Communication {
	return x.wrapResponse(x.MakeResponse(t, status))
}

func (x *Master) Respond(resp *task.Response) {
	c := x.wrapResponse(resp)

	x.Send(c)
}

func (x *Master) Challenge(t *task.Task, challenge string) {
	c := x.MakeCommunicationResponse(t, task.Response_Challenge)
	c.Response.Challenge = &challenge

	x.Send(c)
}
