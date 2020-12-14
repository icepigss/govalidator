package govalidator

// ref https://github.com/go-validator/validator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	defaultValidator = &Validator{
		tagName: "valid",
		validateFuncs: map[string]ValidateFunc{
			"nonzero": nonzero,
			"len":     length,
			"min":     min,
			"max":     max,
			"regex":   regex,
			"nonnil":  nonnil,
			"enum":    enum,
		},
		errMap: map[string]ErrRuleMap{},
	}

	ErrNotSuport      = errors.New("unsuport validate type")
	ErrZeroValue      = errors.New("not allowed zero")
	ErrMin            = errors.New("less than min")
	ErrMax            = errors.New("greater than max")
	ErrLen            = errors.New("invalid length")
	ErrRegexp         = errors.New("regular expression mismatch")
	ErrUnsupported    = errors.New("unsupported type")
	ErrBadParameter   = errors.New("bad parameter")
	ErrUnknownTag     = errors.New("unknown tag")
	ErrInvalid        = errors.New("invalid value")
	ErrCannotValidate = errors.New("cannot validate unexported struct")
	ErrEnum           = errors.New("not allowed out of enum value")
)

type E struct {
	Field string
	Rule  string
	Msg   string
}

type ErrRuleMap map[string]string

type Validator struct {
	tagName       string
	validateFuncs map[string]ValidateFunc
	errMap        map[string]ErrRuleMap
}

type ValidateFunc func(interface{}, string) error

type Error map[string]error

func SetErr(le []E) {
	defaultValidator.SetErr(le)
}

func SetFunc(name string, fn ValidateFunc) {
	defaultValidator.SetFunc(name, fn)
}

func SetTagName(tagName string) {
	defaultValidator.SetTagName(tagName)
}

func Validate(v interface{}) (Error, error) {
	return defaultValidator.Validate(v)
}

func (d *Validator) SetErr(le []E) {
	for _, e := range le {
		if _, ok := d.errMap[e.Field]; !ok {
			d.errMap[e.Field] = ErrRuleMap{}
		}
		d.errMap[e.Field][e.Rule] = e.Msg
	}
}

func (d *Validator) SetFunc(name string, fn ValidateFunc) {
	if name == "" {
		return
	}
	if fn == nil {
		delete(d.validateFuncs, name)
		return
	}
	d.validateFuncs[name] = fn
}

func (d *Validator) SetTagName(tagName string) {
	if tagName != "" {
		d.tagName = tagName
	}
}

func (d *Validator) Validate(v interface{}) (Error, error) {
	var err error
	validErrs := make(Error)

	rv := reflect.ValueOf(v)

	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface && !rv.IsNil() {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return validErrs, ErrNotSuport
	}

	for i := 0; i < rv.NumField(); i++ {
		field := rv.Type().Field(i)

		value := rv.FieldByName(field.Name)
		validErr := d.validateField(field, value)
		if validErr != nil {
			validErrs[field.Name] = validErr
		}
	}
	return validErrs, err
}

func (d *Validator) validateField(field reflect.StructField, value reflect.Value) error {
	tag := field.Tag.Get(d.tagName)

	if tag == "" {
		return nil
	}
	rules := strings.Split(tag, ";")

	for _, rule := range rules {
		rule := strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		var ruleName string
		var ruleValue string
		var err error
		pair := strings.Split(rule, "=")
		if len(pair) > 0 {
			ruleName = strings.TrimSpace(pair[0])
		}
		if len(pair) > 1 {
			ruleValue = strings.TrimSpace(pair[1])
		}
		if fn, ok := d.validateFuncs[ruleName]; ok {
			err = fn(value.Interface(), ruleValue)
		}

		if err != nil {
			if definedErrStr, ok := d.errMap[field.Name][ruleName]; ok {
				if strings.Contains(definedErrStr, `%`) {
					definedErrStr = fmt.Sprintf(definedErrStr, ruleValue)
				}
				return errors.New(definedErrStr)
			}
			return err
		}
	}

	return nil
}

func nonzero(v interface{}, param string) error {
	st := reflect.ValueOf(v)
	valid := true
	switch st.Kind() {
	case reflect.String:
		valid = utf8.RuneCountInString(st.String()) != 0
	case reflect.Ptr, reflect.Interface:
		valid = !st.IsNil()
	case reflect.Slice, reflect.Map, reflect.Array:
		valid = st.Len() != 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		valid = st.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		valid = st.Uint() != 0
	case reflect.Float32, reflect.Float64:
		valid = st.Float() != 0
	case reflect.Bool:
		valid = st.Bool()
	case reflect.Invalid:
		valid = false
	case reflect.Struct:
		valid = true
	default:
	}

	if !valid {
		return ErrZeroValue
	}
	return nil
}

// length tests whether a variable's length is equal to a given
// value. For strings it tests the number of characters whereas
// for maps and slices it tests the number of items.
func length(v interface{}, param string) error {
	st := reflect.ValueOf(v)
	valid := true
	if st.Kind() == reflect.Ptr {
		if st.IsNil() {
			return nil
		}
		st = st.Elem()
	}
	switch st.Kind() {
	case reflect.String:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		valid = int64(utf8.RuneCountInString(st.String())) == p
	case reflect.Slice, reflect.Map, reflect.Array:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		valid = int64(st.Len()) == p
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		valid = st.Int() == p
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p, err := asUint(param)
		if err != nil {
			return ErrBadParameter
		}
		valid = st.Uint() == p
	case reflect.Float32, reflect.Float64:
		p, err := asFloat(param)
		if err != nil {
			return ErrBadParameter
		}
		valid = st.Float() == p
	default:
		return ErrUnsupported
	}
	if !valid {
		return ErrLen
	}
	return nil
}

// min tests whether a variable value is larger or equal to a given
// number. For number types, it's a simple lesser-than test; for
// strings it tests the number of characters whereas for maps
// and slices it tests the number of items.
func min(v interface{}, param string) error {
	st := reflect.ValueOf(v)
	invalid := false
	if st.Kind() == reflect.Ptr {
		if st.IsNil() {
			return nil
		}
		st = st.Elem()
	}
	switch st.Kind() {
	case reflect.String:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = int64(utf8.RuneCountInString(st.String())) < p
	case reflect.Slice, reflect.Map, reflect.Array:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = int64(st.Len()) < p
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Int() < p
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p, err := asUint(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Uint() < p
	case reflect.Float32, reflect.Float64:
		p, err := asFloat(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Float() < p
	default:
		return ErrUnsupported
	}
	if invalid {
		return ErrMin
	}
	return nil
}

// max tests whether a variable value is lesser than a given
// value. For numbers, it's a simple lesser-than test; for
// strings it tests the number of characters whereas for maps
// and slices it tests the number of items.
func max(v interface{}, param string) error {
	st := reflect.ValueOf(v)
	var invalid bool
	if st.Kind() == reflect.Ptr {
		if st.IsNil() {
			return nil
		}
		st = st.Elem()
	}
	switch st.Kind() {
	case reflect.String:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = int64(utf8.RuneCountInString(st.String())) > p
	case reflect.Slice, reflect.Map, reflect.Array:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = int64(st.Len()) > p
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Int() > p
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p, err := asUint(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Uint() > p
	case reflect.Float32, reflect.Float64:
		p, err := asFloat(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Float() > p
	default:
		return ErrUnsupported
	}
	if invalid {
		return ErrMax
	}
	return nil
}

// regex is the builtin validation function that checks
// whether the string variable matches a regular expression
func regex(v interface{}, param string) error {
	s, ok := v.(string)
	if !ok {
		sptr, ok := v.(*string)
		if !ok {
			return ErrUnsupported
		}
		if sptr == nil {
			return nil
		}
		s = *sptr
	}

	re, err := regexp.Compile(param)
	if err != nil {
		return ErrBadParameter
	}

	if !re.MatchString(s) {
		return ErrRegexp
	}
	return nil
}

// asInt retuns the parameter as a int64
// or panics if it can't convert
func asInt(param string) (int64, error) {
	i, err := strconv.ParseInt(param, 0, 64)
	if err != nil {
		return 0, ErrBadParameter
	}
	return i, nil
}

func asIntSlice(items []string) ([]int64, error) {
	rsp := make([]int64, 0, len(items))
	for _, str := range items {
		str = strings.TrimSpace(str)
		i, err := strconv.ParseInt(str, 0, 64)
		if err != nil {
			return rsp, ErrBadParameter
		}
		rsp = append(rsp, i)
	}
	return rsp, nil
}

func trimStringSlice(items []string) ([]string, error) {
	rsp := make([]string, 0, len(items))
	for _, str := range items {
		str = strings.TrimSpace(str)
		rsp = append(rsp, str)
	}
	return rsp, nil
}

// asUint retuns the parameter as a uint64
// or panics if it can't convert
func asUint(param string) (uint64, error) {
	i, err := strconv.ParseUint(param, 0, 64)
	if err != nil {
		return 0, ErrBadParameter
	}
	return i, nil
}

func asUintSlice(items []string) ([]uint64, error) {
	rsp := make([]uint64, 0, len(items))
	for _, str := range items {
		str = strings.TrimSpace(str)
		i, err := strconv.ParseUint(str, 0, 64)
		if err != nil {
			return rsp, ErrBadParameter
		}
		rsp = append(rsp, i)
	}
	return rsp, nil
}

// asFloat retuns the parameter as a float64
// or panics if it can't convert
func asFloat(param string) (float64, error) {
	i, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return 0.0, ErrBadParameter
	}
	return i, nil
}

func asFloatSlice(items []string) ([]float64, error) {
	rsp := make([]float64, 0, len(items))
	for _, str := range items {
		str = strings.TrimSpace(str)
		i, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return rsp, ErrBadParameter
		}
		rsp = append(rsp, i)
	}
	return rsp, nil
}

func inInt64Slice(key int64, match []int64) bool {
	for _, val := range match {
		if key == val {
			return true
		}
	}
	return false
}

func inUintSlice(key uint64, match []uint64) bool {
	for _, val := range match {
		if key == val {
			return true
		}
	}
	return false
}

func inFloatSlice(key float64, match []float64) bool {
	for _, val := range match {
		if key == val {
			return true
		}
	}
	return false
}

func inStringSlice(key string, match []string) bool {
	for _, val := range match {
		if key == val {
			return true
		}
	}
	return false
}

// nonnil validates that the given pointer is not nil
func nonnil(v interface{}, param string) error {
	st := reflect.ValueOf(v)
	// if we got a non-pointer then we most likely got
	// the value for a pointer field, either way, its not
	// nil
	switch st.Kind() {
	case reflect.Ptr, reflect.Interface:
		if st.IsNil() {
			return ErrZeroValue
		}
	case reflect.Invalid:
		// the only way its invalid is if its an interface that's nil
		return ErrZeroValue
	}
	return nil
}

func enum(v interface{}, param string) error {
	items := strings.Split(param, ",")

	st := reflect.ValueOf(v)
	invalid := false
	if st.Kind() == reflect.Ptr {
		if st.IsNil() {
			return nil
		}
		st = st.Elem()
	}
	switch st.Kind() {
	case reflect.String:
		p, err := trimStringSlice(items)
		if err != nil {
			return ErrBadParameter
		}
		invalid = !inStringSlice(st.String(), p)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err := asIntSlice(items)
		if err != nil {
			return ErrBadParameter
		}
		invalid = !inInt64Slice(st.Int(), p)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p, err := asUintSlice(items)
		if err != nil {
			return ErrBadParameter
		}
		invalid = !inUintSlice(st.Uint(), p)
	case reflect.Float32, reflect.Float64:
		p, err := asFloatSlice(items)
		if err != nil {
			return ErrBadParameter
		}
		invalid = !inFloatSlice(st.Float(), p)
	default:
		return ErrUnsupported
	}
	if invalid {
		return ErrEnum
	}
	return nil
}
