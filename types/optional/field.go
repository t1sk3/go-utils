package optional

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Field is a value in a JSON document that tells three cases apart: the field was left
// out, it was null, or it had a value. Optional cannot, because both of the first two
// leave it empty.
//
// That difference is what a partial update needs. In a PATCH request, leaving a field
// out means "keep it" and null means "clear it":
//
//	type UpdateUser struct {
//		Name  optional.Field[string] `json:"name,omitzero"`
//		Email optional.Field[string] `json:"email,omitzero"`
//	}
//
//	if req.Email.IsNull() {
//		user.Email = nil
//	} else if v, ok := req.Email.Value(); ok {
//		user.Email = &v
//	}
//
// The zero value is an absent Field. Tag fields with omitzero so that an absent Field
// is left out when encoding too; without it, it is written as null.
type Field[T any] struct {
	value T
	state fieldState
}

type fieldState uint8

const (
	fieldAbsent fieldState = iota
	fieldNull
	fieldValue
)

// FieldOf returns a Field holding v.
func FieldOf[T any](v T) Field[T] {
	return Field[T]{value: v, state: fieldValue}
}

// NullField returns a Field that is explicitly null.
func NullField[T any]() Field[T] {
	return Field[T]{state: fieldNull}
}

// IsAbsent reports whether the field was left out.
func (f Field[T]) IsAbsent() bool {
	return f.state == fieldAbsent
}

// IsNull reports whether the field was explicitly null.
func (f Field[T]) IsNull() bool {
	return f.state == fieldNull
}

// HasValue reports whether the field had a value.
func (f Field[T]) HasValue() bool {
	return f.state == fieldValue
}

// Value returns the value and whether there is one. It is false for both an absent and
// a null Field.
func (f Field[T]) Value() (T, bool) {
	return f.value, f.state == fieldValue
}

// Optional returns the value as an Optional, which is empty for both an absent and a
// null Field.
func (f Field[T]) Optional() Optional[T] {
	return OfOk(f.value, f.state == fieldValue)
}

// IsZero reports whether the field is absent. It lets encoding/json leave an absent
// Field out when it is tagged with `json:",omitzero"`.
func (f Field[T]) IsZero() bool {
	return f.state == fieldAbsent
}

// String formats f as "Field[value]", "Field.null" or "Field.absent".
func (f Field[T]) String() string {
	switch f.state {
	case fieldValue:
		return fmt.Sprintf("Field[%v]", f.value)
	case fieldNull:
		return "Field.null"
	}
	return "Field.absent"
}

// MarshalJSON encodes the value, or null for a null or absent Field. To leave an absent
// Field out instead, tag it with omitzero.
func (f Field[T]) MarshalJSON() ([]byte, error) {
	if f.state != fieldValue {
		return []byte("null"), nil
	}
	return json.Marshal(f.value)
}

// UnmarshalJSON decodes null as a null Field and anything else as a value. A field that
// is missing from the document is never decoded, so it stays absent.
func (f *Field[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		*f = NullField[T]()
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*f = FieldOf(v)
	return nil
}
