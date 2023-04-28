package comm

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/andyleap/nostr/proto"
)

var (
	ErrInvalidComm = errors.New("invalid comm")
)

type Req interface {
	MarshalJSON() ([]byte, error)
	req()
}

type Publish struct {
	Event *proto.Event
}

func (p *Publish) req() {}

func (p *Publish) MarshalJSON() ([]byte, error) {
	msg := []interface{}{
		"EVENT",
		p.Event,
	}
	return json.Marshal(msg)
}

type Subscribe struct {
	ID      string
	Filters []*Filter
}

func (s *Subscribe) req() {}

func (s *Subscribe) MarshalJSON() ([]byte, error) {
	msg := []interface{}{
		"REQ",
		s.ID,
	}
	for _, f := range s.Filters {
		msg = append(msg, f)
	}
	return json.Marshal(msg)
}

type Close struct {
	ID string
}

func (c *Close) req() {}

func (c *Close) MarshalJSON() ([]byte, error) {
	msg := []interface{}{
		"CLOSE",
		c.ID,
	}
	return json.Marshal(msg)
}

func ParseReq(data []byte) (Req, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	start, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if start != json.Delim('[') {
		return nil, ErrInvalidComm
	}
	var t string
	err = dec.Decode(&t)
	if err != nil {
		return nil, err
	}
	var req Req
	switch t {
	case "EVENT":
		p := &Publish{}
		err = dec.Decode(&p.Event)
		if err != nil {
			return nil, err
		}
		req = p
	case "REQ":
		s := &Subscribe{}
		err = dec.Decode(&s.ID)
		if err != nil {
			return nil, err
		}
		for dec.More() {
			var f Filter
			err = dec.Decode(&f)
			if err != nil {
				return nil, err
			}
			s.Filters = append(s.Filters, &f)
		}
		req = s
	case "CLOSE":
		c := &Close{}
		err = dec.Decode(&c.ID)
		if err != nil {
			return nil, err
		}
		req = c
	default:
		return nil, ErrInvalidComm
	}
	return req, nil
}

type Resp interface {
	MarshalJSON() ([]byte, error)
	resp()
}

type Event struct {
	ID    string
	Event *proto.Event
}

func (e *Event) resp() {}

func (e *Event) MarshalJSON() ([]byte, error) {
	msg := []interface{}{
		"EVENT",
		e.ID,
		e.Event,
	}
	return json.Marshal(msg)
}

type EndOfStoredEvents struct {
	ID string
}

func (e *EndOfStoredEvents) resp() {}

func (e *EndOfStoredEvents) MarshalJSON() ([]byte, error) {
	msg := []interface{}{
		"EOSE",
		e.ID,
	}
	return json.Marshal(msg)
}

type Notice struct {
	Msg string
}

func (n *Notice) resp() {}

func (n *Notice) MarshalJSON() ([]byte, error) {
	msg := []interface{}{
		"NOTICE",
		n.Msg,
	}
	return json.Marshal(msg)
}

func ParseResp(data []byte) (Resp, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	start, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if start != json.Delim('[') {
		return nil, ErrInvalidComm
	}
	var t string
	err = dec.Decode(&t)
	if err != nil {
		return nil, err
	}
	var resp Resp
	switch t {
	case "EVENT":
		e := &Event{}
		err = dec.Decode(&e.ID)
		if err != nil {
			return nil, err
		}
		err = dec.Decode(&e.Event)
		if err != nil {
			return nil, err
		}
		resp = e
	case "EOSE":
		e := &EndOfStoredEvents{}
		err = dec.Decode(&e.ID)
		if err != nil {
			return nil, err
		}
		resp = e
	case "NOTICE":
		n := &Notice{}
		err = dec.Decode(&n.Msg)
		if err != nil {
			return nil, err
		}
		resp = n
	default:
		return nil, ErrInvalidComm
	}
	return resp, nil
}
