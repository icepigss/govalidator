# govalidator
A simple validator that can customize error messages.
E.g
* you can use it to verify parameters and return error message in restful api.
### Example
```Golang
type User struct {
	Name string   `vd:"nonzero;len=7;mycheck"`
	Age  int      `vd:"nonzero;max=24"`
	Pics []string `vd:"min=1"`
}


user := User{}
user.Name = "icepigss"
user.Age = 27

// customize tag name, default is "validate"
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
```
