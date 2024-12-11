package rawjson

import "encoding/json"

type RJSON struct {
	data interface{}
}

func Parse(body []byte) (*RJSON, error) {
	var raw interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	return New(raw), nil
}

func New(data interface{}) *RJSON {
	return &RJSON{data: data}
}

func (r *RJSON) Map() map[string]interface{} {
	return r.data.(map[string]interface{})
}

func (r *RJSON) Get(key string) *RJSON {
	m := r.Map()
	return New(m[key])
}

func (r *RJSON) Array() []interface{} {
	return r.data.([]interface{})
}

func (r *RJSON) GetIndex(index int) *RJSON {
	a := r.Array()
	return New(a[index])
}

func (r *RJSON) Len() int {
	return len(r.Array())
}

func (r *RJSON) String() string {
	return r.data.(string)
}
