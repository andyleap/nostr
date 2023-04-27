package comm

import (
	"encoding/json"

	"github.com/andyleap/nostr/proto"
)

type Filter struct {
	IDs     []string
	Authors []string
	Kinds   []int64
	Since   int64
	Until   int64
	Limit   int64

	TagFilters map[string][]string
}

func (f *Filter) MarshalJSON() ([]byte, error) {
	msg := make(map[string]interface{})
	if len(f.IDs) > 0 {
		msg["ids"] = f.IDs
	}
	if len(f.Authors) > 0 {
		msg["authors"] = f.Authors
	}
	if len(f.Kinds) > 0 {
		msg["kinds"] = f.Kinds
	}
	if f.Since > 0 {
		msg["since"] = f.Since
	}
	if f.Until > 0 {
		msg["until"] = f.Until
	}
	if f.Limit > 0 {
		msg["limit"] = f.Limit
	}
	for k, v := range f.TagFilters {
		msg["#"+k] = v
	}
	return json.Marshal(msg)
}

func (f *Filter) UnmarshalJSON(data []byte) error {
	msg := map[string]*json.RawMessage{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return err
	}
	f.TagFilters = map[string][]string{}
	for k, v := range msg {
		switch k {
		case "ids":
			json.Unmarshal(*v, &f.IDs)
		case "authors":
			json.Unmarshal(*v, &f.Authors)
		case "kinds":
			json.Unmarshal(*v, &f.Kinds)
		case "since":
			json.Unmarshal(*v, &f.Since)
		case "until":
			json.Unmarshal(*v, &f.Until)
		case "limit":
			json.Unmarshal(*v, &f.Limit)
		default:
			if k[0] == '#' {
				var vs []string
				json.Unmarshal(*v, &vs)
				f.TagFilters[k[1:]] = vs
			}
		}
	}
	return nil
}

func (f *Filter) Match(e *proto.Event) bool {
	if len(f.IDs) > 0 && !contains(f.IDs, e.ID) {
		return false
	}
	if len(f.Authors) > 0 && !contains(f.Authors, e.PubKey) {
		return false
	}
	if len(f.Kinds) > 0 && !contains(f.Kinds, e.Kind) {
		return false
	}
	if f.Since > 0 && e.CreatedAt < f.Since {
		return false
	}
	if f.Until > 0 && e.CreatedAt > f.Until {
		return false
	}
	if len(f.TagFilters) > 0 {
		for k, v := range f.TagFilters {
			match := false
			for _, t := range e.Tags {
				if t[0] == k && !contains(v, t[1]) {
					match = true
					break
				}
			}
			if !match {
				return false
			}
		}
	}

	return true
}

func contains[t comparable](s []t, e t) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
