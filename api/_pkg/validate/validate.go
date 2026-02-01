package validate

import (
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	instance *validator.Validate
	once     sync.Once
)

func Get() *validator.Validate {
	once.Do(func() {
		instance = validator.New(validator.WithRequiredStructEnabled())
		instance.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
	})
	return instance
}

func Struct(s interface{}) error {
	return Get().Struct(s)
}

func FormatError(err error) string {
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return err.Error()
	}
	msgs := make([]string, 0, len(ve))
	for _, fe := range ve {
		msgs = append(msgs, formatFieldError(fe))
	}
	return strings.Join(msgs, "; ")
}

func formatFieldError(fe validator.FieldError) string {
	field := fe.Field()
	switch fe.Tag() {
	case "required":
		return field + " is required"
	case "oneof":
		values := strings.Split(fe.Param(), " ")
		quoted := make([]string, len(values))
		for i, v := range values {
			quoted[i] = "'" + v + "'"
		}
		if len(quoted) > 1 {
			return field + " must be " + strings.Join(quoted[:len(quoted)-1], ", ") + ", or " + quoted[len(quoted)-1]
		}
		return field + " must be " + quoted[0]
	case "gte":
		if fe.Param() == "0" {
			return field + " must be non-negative"
		}
		return field + " must be at least " + fe.Param()
	case "lte":
		return field + " must not exceed " + fe.Param()
	default:
		return field + " is invalid"
	}
}
