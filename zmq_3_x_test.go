// +build zmq_3_x

/*
  Copyright 2010 Alec Thomas

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
*/
package gozmq

import (
	"testing"
	"time"
)

const ADDR_PROXY_IN = "tcp://127.0.0.1:24114"
const ADDR_PROXY_OUT = "tcp://127.0.0.1:24115"
const ADDR_PROXY_CAP = "tcp://127.0.0.1:24116"

func TestProxy(t *testing.T) {
	te1, te2 := NewTestEnv(t), NewTestEnv(t)
	exitOk := make(chan bool, 1)
	go func() {
		in := te1.NewBoundSocket(ROUTER, ADDR_PROXY_IN)
		out := te1.NewBoundSocket(DEALER, ADDR_PROXY_OUT)
		capture := te1.NewBoundSocket(PUSH, ADDR_PROXY_CAP)
		err := Proxy(in, out, capture)

		select {
		case <-exitOk:
		default:
			t.Error("Proxy() failed: ", err)
		}
	}()

	in := te2.NewConnectedSocket(REQ, ADDR_PROXY_IN)
	out := te2.NewConnectedSocket(REP, ADDR_PROXY_OUT)
	capture := te2.NewConnectedSocket(PULL, ADDR_PROXY_CAP)
	time.Sleep(1e8)
	te2.Send(in, nil, 0)
	te2.Recv(out, 0)
	te2.Recv(capture, 0)

	te2.Close()
	exitOk <- true
	te1.Close()
}

func TestProxyNoCapture(t *testing.T) {
	te1, te2 := NewTestEnv(t), NewTestEnv(t)
	exitOk := make(chan bool, 1)
	go func() {
		in := te1.NewBoundSocket(ROUTER, ADDR_PROXY_IN)
		out := te1.NewBoundSocket(DEALER, ADDR_PROXY_OUT)
		err := Proxy(in, out, nil)

		select {
		case <-exitOk:
		default:
			t.Error("Proxy() failed: ", err)
		}
	}()

	in := te2.NewConnectedSocket(REQ, ADDR_PROXY_IN)
	out := te2.NewConnectedSocket(REP, ADDR_PROXY_OUT)
	time.Sleep(1e8)
	te2.Send(in, nil, 0)
	te2.Recv(out, 0)

	te2.Close()
	exitOk <- true
	te1.Close()
}

func TestSocket_SetSockOptStringNil(t *testing.T) {
	failed := make(chan bool, 2)
	c, _ := NewContext()
	defer c.Close()
	go func() {
		srv, _ := c.NewSocket(REP)
		defer srv.Close()
		srv.SetSockOptString(TCP_ACCEPT_FILTER, "127.0.0.1")
		srv.SetSockOptString(TCP_ACCEPT_FILTER, "192.0.2.1")
		srv.Bind(ADDRESS1) // 127.0.0.1 and 192.0.2.1 are allowed here.
		// The test will fail if the following line is removed:
		srv.SetSockOptStringNil(TCP_ACCEPT_FILTER)
		srv.SetSockOptString(TCP_ACCEPT_FILTER, "192.0.2.2")
		srv.Bind(ADDRESS2) // Only 192.0.2.1 is allowed here.
		for {
			if _, err := srv.Recv(0); err != nil {
				break
			}
			srv.Send(nil, 0)
		}
	}()
	go func() {
		s2, _ := c.NewSocket(REQ)
		defer s2.Close()
		s2.SetSockOptInt(LINGER, 0)
		s2.Connect(ADDRESS2)
		s2.Send(nil, 0)
		if _, err := s2.Recv(0); err == nil {
			// 127.0.0.1 is supposed to be ignored by ADDRESS2:
			t.Error("SetSockOptStringNil did not clear TCP_ACCEPT_FILTER.")
		}
		failed <- true
	}()
	s1, _ := c.NewSocket(REQ)
	defer s1.Close()
	s1.Connect(ADDRESS1)
	s1.Send(nil, 0)
	s1.Recv(0)
	select {
	case <-failed:
	case <-time.After(50 * time.Millisecond):
	}
}
