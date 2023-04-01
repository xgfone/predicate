// Copyright 2022~2023 xgfone
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
	"strings"
)

const (
	and = "&&"
	or  = "||"
	neq = "!="
	eq  = "=="
	le  = "<="
	ge  = ">="
	lt  = "<"
	gt  = ">"
)

type tree struct {
	builder   *Builder
	matcher   string
	bfunc     BuilderFunction
	values    []interface{}
	not       bool
	ruleLeft  *tree
	ruleRight *tree
}

type treeBuilder func() *tree

func (tb treeBuilder) Build(c BuilderContext) error {
	rule := tb()
	return rule.builder.buildRuleOnAnd(c, rule)
}

// ContextBuilder is a context builder interface.
type ContextBuilder interface {
	Build(BuilderContext) error
}

// BuilderContext is the builder context.
type BuilderContext interface {
	New() BuilderContext
	Not(BuilderContext)
	And(BuilderContext)
	Or(BuilderContext)
}

// BuilderFunction represents a function to be called and build the context.
type BuilderFunction func(context BuilderContext, args ...interface{}) error

// BinaryBuilderFunc is the builder function of the binary operator.
type BinaryBuilderFunc func(ctx BuilderContext, left, right interface{}) error

// Builder is a simple rule builder to parse the rule and build the context.
type Builder struct {
	GetIdentifier GetIdentifierFn
	GetProperty   GetPropertyFn

	NEQ BinaryBuilderFunc
	EQ  BinaryBuilderFunc
	LT  BinaryBuilderFunc
	GT  BinaryBuilderFunc
	LE  BinaryBuilderFunc
	GE  BinaryBuilderFunc

	parser Parser
	pfuncs map[string]interface{}
	funcs  map[string]BuilderFunction
}

// NewBuilder returns a new rule builder.
func NewBuilder() *Builder {
	buidler := &Builder{
		pfuncs: make(map[string]interface{}),
		funcs:  make(map[string]BuilderFunction),
	}

	buidler.parser, _ = NewParser(Def{
		GetIdentifier: buidler.getIdentifier,
		GetProperty:   buidler.getProperty,
		Functions:     buidler.pfuncs,
		Operators: Operators{
			EQ:  buidler.eqOp,
			NEQ: buidler.neqOp,
			LT:  buidler.ltOp,
			GT:  buidler.gtOp,
			LE:  buidler.leOp,
			GE:  buidler.geOp,
			OR:  buidler.orOp,
			AND: buidler.andOp,
			NOT: buidler.notOp,
		},
	})
	return buidler
}

// GetFunc returns the registered builder function.
//
// If the function is not registered, return nil.
func (b *Builder) GetFunc(name string) BuilderFunction {
	return b.funcs[name]
}

// RegisterFunc registers the builder function with the name.
//
// If the function name has existed, reset it to the new function.
func (b *Builder) RegisterFunc(name string, f BuilderFunction) {
	if name == "" {
		panic("the builder function name must be empty")
	}
	if f == nil {
		panic("the builder function must not be nil")
	}

	b.funcs[name] = f
	b.pfuncs[name] = func(values ...interface{}) treeBuilder {
		return func() *tree { return &tree{builder: b, matcher: name, values: values} }
	}
}

// GetAllFuncNames returns the names of all the registered builder functions.
func (b *Builder) GetAllFuncNames() (names []string) {
	names = make([]string, 0, len(b.funcs))
	for name := range b.funcs {
		names = append(names, name)
	}
	return
}

func (b *Builder) getIdentifier(selector []string) (interface{}, error) {
	if b.GetIdentifier == nil {
		return nil, fmt.Errorf("%s is not defined", strings.Join(selector, "."))
	}

	ident, err := b.GetIdentifier(selector)
	if err != nil {
		return nil, err
	}

	if f, ok := ident.(BuilderFunction); ok {
		ident = treeBuilder(func() *tree { return &tree{builder: b, bfunc: f} })
	}

	return ident, nil
}

func (b *Builder) getProperty(mapVal, keyVal interface{}) (interface{}, error) {
	if b.GetProperty == nil {
		return nil, fmt.Errorf("properties are not supported")
	}
	return b.GetProperty(mapVal, keyVal)
}

func (b *Builder) neqOp(left, right interface{}) treeBuilder {
	return func() *tree {
		return &tree{
			builder: b,
			matcher: neq,
			values:  []interface{}{left, right},
		}
	}
}

func (b *Builder) eqOp(left, right interface{}) treeBuilder {
	return func() *tree {
		return &tree{
			builder: b,
			matcher: eq,
			values:  []interface{}{left, right},
		}
	}
}

func (b *Builder) ltOp(left, right interface{}) treeBuilder {
	return func() *tree {
		return &tree{
			builder: b,
			matcher: lt,
			values:  []interface{}{left, right},
		}
	}
}

func (b *Builder) gtOp(left, right interface{}) treeBuilder {
	return func() *tree {
		return &tree{
			builder: b,
			matcher: gt,
			values:  []interface{}{left, right},
		}
	}
}

func (b *Builder) leOp(left, right interface{}) treeBuilder {
	return func() *tree {
		return &tree{
			builder: b,
			matcher: le,
			values:  []interface{}{left, right},
		}
	}
}

func (b *Builder) geOp(left, right interface{}) treeBuilder {
	return func() *tree {
		return &tree{
			builder: b,
			matcher: ge,
			values:  []interface{}{left, right},
		}
	}
}

func (b *Builder) orOp(left, right treeBuilder) treeBuilder {
	return func() *tree {
		return &tree{
			builder:   b,
			matcher:   or,
			ruleLeft:  left(),
			ruleRight: right(),
		}
	}
}

func (b *Builder) andOp(left, right treeBuilder) treeBuilder {
	return func() *tree {
		return &tree{
			builder:   b,
			matcher:   and,
			ruleLeft:  left(),
			ruleRight: right(),
		}
	}
}

func (b *Builder) notOp(elem treeBuilder) treeBuilder {
	return func() *tree { return invert(elem()) }
}

func invert(t *tree) *tree {
	switch t.matcher {
	case or:
		t.matcher = and
		t.ruleLeft = invert(t.ruleLeft)
		t.ruleRight = invert(t.ruleRight)
	case and:
		t.matcher = or
		t.ruleLeft = invert(t.ruleLeft)
		t.ruleRight = invert(t.ruleRight)
	default:
		t.not = !t.not
	}
	return t
}

// Build parses the rule and build the result into the context..
func (b *Builder) Build(c BuilderContext, rule string) (err error) {
	// Parse the input rule into an AST.
	parse, err := b.parser.Parse(rule)
	if err != nil {
		return err
	}

	// Traverse the AST to run the predicate functions.
	switch f := parse.(type) {
	case treeBuilder:
		err = b.buildRuleOnAnd(c, f())
	case BuilderFunction:
		err = f(c)
	default:
		err = fmt.Errorf("unexpected parsed type '%T'", parse)
	}

	return
}

func checkRule(rule *tree) error {
	if len(rule.values) == 0 {
		return fmt.Errorf("no args for the rule '%s'", rule.matcher)
	}
	return nil
}

func (b *Builder) buildRuleOnAnd(ctx BuilderContext, rule *tree) (err error) {
	switch rule.matcher {
	case and:
		if err = b.buildRuleOnAnd(ctx, rule.ruleLeft); err != nil {
			return
		}
		return b.buildRuleOnAnd(ctx, rule.ruleRight)

	case or:
		newctx := ctx.New()
		if err = b.buildRuleOnOr(newctx, rule.ruleLeft); err != nil {
			return
		}
		if err = b.buildRuleOnOr(newctx, rule.ruleRight); err != nil {
			return
		}

		ctx.Or(newctx)
		return

	case neq:
		return b.buildBinaryRule(ctx, b.NEQ, neq, rule.values)

	case eq:
		return b.buildBinaryRule(ctx, b.EQ, eq, rule.values)

	case le:
		return b.buildBinaryRule(ctx, b.LE, le, rule.values)

	case ge:
		return b.buildBinaryRule(ctx, b.GE, ge, rule.values)

	case lt:
		return b.buildBinaryRule(ctx, b.LT, lt, rule.values)

	case gt:
		return b.buildBinaryRule(ctx, b.GT, gt, rule.values)

	case "":
		return rule.bfunc(ctx)

	default:
		if err = checkRule(rule); err != nil {
			return err
		}

		f := b.funcs[rule.matcher]
		if f == nil {
			return fmt.Errorf("the function '%s' is unsupported", rule.matcher)
		}

		if rule.not {
			f = b.buildRuleOnNot(f)
		}
		return f(ctx, rule.values...)
	}
}

func (b *Builder) buildRuleOnOr(ctx BuilderContext, rule *tree) (err error) {
	switch rule.matcher {
	case and:
		newctx := ctx.New()
		if err = b.buildRuleOnAnd(newctx, rule.ruleLeft); err != nil {
			return
		}
		if err = b.buildRuleOnAnd(newctx, rule.ruleRight); err != nil {
			return
		}

		ctx.And(newctx)
		return

	case or:
		if err = b.buildRuleOnOr(ctx, rule.ruleLeft); err != nil {
			return
		}

		return b.buildRuleOnOr(ctx, rule.ruleRight)

	case neq:
		return b.buildBinaryRule(ctx, b.NEQ, neq, rule.values)

	case eq:
		return b.buildBinaryRule(ctx, b.EQ, eq, rule.values)

	case le:
		return b.buildBinaryRule(ctx, b.LE, le, rule.values)

	case ge:
		return b.buildBinaryRule(ctx, b.GE, ge, rule.values)

	case lt:
		return b.buildBinaryRule(ctx, b.LT, lt, rule.values)

	case gt:
		return b.buildBinaryRule(ctx, b.GT, gt, rule.values)

	case "":
		return rule.bfunc(ctx)

	default:
		if err = checkRule(rule); err != nil {
			return
		}

		f := b.funcs[rule.matcher]
		if f == nil {
			return fmt.Errorf("the function '%s' is unsupported", rule.matcher)
		}

		if rule.not {
			f = b.buildRuleOnNot(f)
		}

		newctx := ctx.New()
		err = f(newctx, rule.values...)
		if err == nil {
			ctx.And(newctx)
		}

		return
	}
}

func (b *Builder) buildBinaryRule(ctx BuilderContext, f BinaryBuilderFunc,
	op string, vs []interface{}) error {
	if f == nil {
		return fmt.Errorf("no binary %s operator function is set", op)
	}

	if b, ok := vs[0].(treeBuilder); ok {
		left := b()
		if left.bfunc != nil {
			vs[0] = left.bfunc
		}
	}

	if b, ok := vs[1].(treeBuilder); ok {
		left := b()
		if left.bfunc != nil {
			vs[1] = left.bfunc
		}
	}

	return f(ctx, vs[0], vs[1])
}

func (b *Builder) buildRuleOnNot(f BuilderFunction) BuilderFunction {
	return func(ctx BuilderContext, args ...interface{}) (err error) {
		newctx := ctx.New()
		err = f(newctx, args...)
		if err == nil {
			ctx.Not(newctx)
		}
		return
	}
}
