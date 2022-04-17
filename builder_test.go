// Copyright 2022 xgfone
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package predicate

import (
	"fmt"
)

// ************************************************************************* //
/// Define some validators.

type Validator interface {
	Validate(interface{}) error
}

type ValidatorFunc func(interface{}) error

func (v ValidatorFunc) Validate(i interface{}) error { return v(i) }

type AndValidator []Validator

func (vs AndValidator) Validate(i interface{}) (err error) {
	for _, v := range vs {
		if err = v.Validate(i); err != nil {
			return
		}
	}
	return
}

type OrValidator []Validator

func (vs OrValidator) Validate(i interface{}) (err error) {
	for _, v := range vs {
		if err = v.Validate(i); err == nil {
			return nil
		}
	}
	return
}

func ZeroValidator() Validator {
	return ValidatorFunc(func(i interface{}) error {
		switch t := i.(type) {
		case string:
			if len(t) > 0 {
				return fmt.Errorf("the string is not empty")
			}

		case int:
			if t != 0 {
				return fmt.Errorf("the integer is not equal to 0")
			}

		default:
			return fmt.Errorf("unsupported type '%T'", i)
		}

		return nil
	})
}

func MinValidator(i int) Validator {
	return ValidatorFunc(func(v interface{}) error {
		switch t := v.(type) {
		case int:
			if t < i {
				return fmt.Errorf("the integer is less than %d", i)
			}

		case string:
			if len(t) < i {
				return fmt.Errorf("the string length is less than %d", i)
			}

		default:
			return fmt.Errorf("unsupported type '%T'", v)
		}

		return nil
	})
}

func MaxValidator(i int) Validator {
	return ValidatorFunc(func(v interface{}) error {
		switch t := v.(type) {
		case int:
			if t > i {
				return fmt.Errorf("the integer is greater than %d", i)
			}

		case string:
			if len(t) > i {
				return fmt.Errorf("the string length is greater than %d", i)
			}

		default:
			return fmt.Errorf("unsupported type '%T'", v)
		}

		return nil
	})
}

// ************************************************************************* //
/// Define validation buidler context.

type context struct {
	validators []Validator
}

func newContext() *context { return &context{} }

func (c *context) New() BuilderContext   { return newContext() }
func (c *context) Not(nc BuilderContext) {}

func (c *context) And(nc BuilderContext) {
	c.AppendValidators(AndValidator(nc.(*context).Validators()))
}

func (c *context) Or(nc BuilderContext) {
	c.AppendValidators(OrValidator(nc.(*context).Validators()))
}

func (c *context) AppendValidators(validators ...Validator) {
	c.validators = append(c.validators, validators...)
}

func (c *context) Validators() []Validator { return c.validators }

func ExampleBuilder() {
	builder := NewBuilder()

	builder.RegisterFunc("zero", func(ctx BuilderContext, args ...interface{}) error {
		ctx.(*context).AppendValidators(ZeroValidator())
		return nil
	})
	builder.RegisterFunc("min", func(ctx BuilderContext, args ...interface{}) error {
		// For simpleness, we don't check whether args is valid.
		ctx.(*context).AppendValidators(MinValidator(args[0].(int)))
		return nil
	})
	builder.RegisterFunc("max", func(ctx BuilderContext, args ...interface{}) error {
		// For simpleness, we don't check whether args is valid.
		ctx.(*context).AppendValidators(MaxValidator(args[0].(int)))
		return nil
	})

	builder.GetIdentifier = func(selector []string) (interface{}, error) {
		// Support the single indentity as the validator,
		// such as "zero" instead of "zero()".
		return builder.GetFunc(selector[0]), nil
	}

	builder.EQ = func(ctx BuilderContext, left, right interface{}) error {
		// Support the format "min == 123" or "123 == min"
		if f, ok := left.(BuilderFunction); ok {
			return f(ctx, right)
		}
		if f, ok := right.(BuilderFunction); ok {
			return f(ctx, left)
		}
		return fmt.Errorf("the left or right is not a BuilderFunction: %T", left)
	}

	ctx := newContext()
	err := builder.Build(ctx, `min(1) && max(10)`) // The function mode
	if err != nil {
		fmt.Println(err)
	} else {
		validator := AndValidator(ctx.Validators())

		fmt.Println(validator.Validate(0))
		fmt.Println(validator.Validate(1))
		fmt.Println(validator.Validate(5))
		fmt.Println(validator.Validate(10))
		fmt.Println(validator.Validate(11))
	}

	ctx = newContext()
	err = builder.Build(ctx, `zero || (min == 5 && 10 == max)`) // The identity+operator mode
	if err != nil {
		fmt.Println(err)
	} else {
		validator := AndValidator(ctx.Validators())

		fmt.Println(validator.Validate(""))
		fmt.Println(validator.Validate("abc"))
		fmt.Println(validator.Validate("abcdefg"))
		fmt.Println(validator.Validate("abcdefghijklmn"))
	}

	// Output:
	// the integer is less than 1
	// <nil>
	// <nil>
	// <nil>
	// the integer is greater than 10
	// <nil>
	// the string length is less than 5
	// <nil>
	// the string length is greater than 10
}
