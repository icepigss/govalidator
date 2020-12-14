package govalidator

import (
	"errors"
	"testing"
)

type User struct {
	Name string   `vd:"nonzero;len=7;mycheck"`
	Age  int      `vd:"nonzero;max=24"`
	Pics []string `vd:"min=1"`
}

func TestValidate(t *testing.T) {
	user := User{}
	user.Name = "icepigss"
	user.Age = 27

	// customize tag name
	SetTagName("vd")

	// customize error message
	SetErr([]E{
		E{"Name", "nonzero", "name must not be empty"},
		E{"Name", "max", "the length of name must be less than %v"},
		E{"Age", "nonzero", "age must not be zero"},
		E{"Age", "max", "age must be less than %v"},
		E{"Pics", "min", "the number of pictures must be more than %v"},
	})

	// customize user verification function
	SetFunc("mycheck", mycheck)

	resp, err := Validate(user)
	t.Logf("resp: %+v, err: %v\n", resp, err)
}

func mycheck(v interface{}, p string) error {
	return errors.New("mycheck error")
}
