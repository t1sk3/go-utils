package optional

import (
	"database/sql"
	"database/sql/driver"
)

// Scan implements sql.Scanner, so an Optional can receive a nullable column
// directly: SQL NULL becomes an empty Optional and anything else a present value.
//
//	var email optional.Optional[string]
//	err := db.QueryRow("SELECT email FROM users WHERE id = $1", id).Scan(&email)
//
// Conversions follow the same rules as Rows.Scan, so T may itself be a type that
// implements sql.Scanner.
func (o *Optional[T]) Scan(src any) error {
	var n sql.Null[T]
	if err := n.Scan(src); err != nil {
		return err
	}
	*o = OfOk(n.V, n.Valid)
	return nil
}

// SQL returns o as a Null, for passing it as a query argument:
//
//	db.Exec("UPDATE users SET email = $1 WHERE id = $2", email.SQL(), id)
func (o Optional[T]) SQL() Null[T] {
	return Null[T]{Optional: o}
}

// Null is an Optional that also implements driver.Valuer, so it can be written to a
// database as well as read from one. Use it for struct fields that are mapped to
// nullable columns by a query builder or ORM.
//
// Null exists because Optional's own Value method returns (T, bool) and so cannot
// also satisfy driver.Valuer. Every other Optional method is available on Null; the
// (T, bool) form is still reachable as n.Optional.Value().
type Null[T any] struct {
	Optional[T]
}

// Value implements driver.Valuer. An empty Null is written as SQL NULL, and a
// present value is converted with driver.DefaultParameterConverter, so named types
// and types that implement driver.Valuer themselves work as they would on their own.
func (n Null[T]) Value() (driver.Value, error) {
	if !n.present {
		return nil, nil
	}
	return driver.DefaultParameterConverter.ConvertValue(n.value)
}
