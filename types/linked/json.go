package linked

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// MarshalJSON encodes m as a JSON object with its entries in insertion order, which a
// built-in map cannot do. Keys must encode as JSON strings or numbers; numbers are
// quoted, as encoding/json does for integer map keys.
func (m *Map[K, V]) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for n := m.first; n != nil; n = n.next {
		if n != m.first {
			b.WriteByte(',')
		}
		key, err := json.Marshal(n.key)
		if err != nil {
			return nil, err
		}
		switch {
		case key[0] == '"':
			b.Write(key)
		case key[0] == '-' || (key[0] >= '0' && key[0] <= '9'):
			b.WriteByte('"')
			b.Write(key)
			b.WriteByte('"')
		default:
			return nil, fmt.Errorf("linked: key %s of type %T cannot be a JSON object key", key, n.key)
		}
		b.WriteByte(':')
		value, err := json.Marshal(n.value)
		if err != nil {
			return nil, err
		}
		b.Write(value)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// UnmarshalJSON decodes a JSON object into m, replacing its contents and keeping the
// order in which the keys appear in the document. If a key appears more than once, the
// last value wins and the key keeps its first position. null leaves m empty.
func (m *Map[K, V]) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	m.Clear()
	if tok == nil {
		return nil
	}
	if tok != json.Delim('{') {
		return errors.New("linked: cannot decode JSON that is not an object into a Map")
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		name := tok.(string) // The decoder only yields strings as object keys.
		var key K
		quoted, _ := json.Marshal(name)
		if err := json.Unmarshal(quoted, &key); err != nil {
			// A numeric key type is written as a quoted number.
			if json.Unmarshal([]byte(name), &key) != nil {
				return fmt.Errorf("linked: cannot decode object key %q into %T", name, key)
			}
		}
		var value V
		if err := dec.Decode(&value); err != nil {
			return err
		}
		m.Put(key, value)
	}
	_, err = dec.Token() // the closing brace
	return err
}

// MarshalJSON encodes s as a JSON array, in insertion order.
func (s *Set[T]) MarshalJSON() ([]byte, error) {
	values := make([]T, 0, s.Len())
	for v := range s.All() {
		values = append(values, v)
	}
	return json.Marshal(values)
}

// UnmarshalJSON decodes a JSON array into s, replacing its contents and keeping the
// order of first occurrence.
func (s *Set[T]) UnmarshalJSON(data []byte) error {
	var values []T
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	s.Clear()
	s.Add(values...)
	return nil
}
