package myrpc

import (
	"fmt"
	"reflect"
	"testing"
)

type Foo int
type Args struct { Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

// it's not a exported Method
func (f Foo) sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func _assert(condition bool, msg string, v ...interface{}) {
	if !condition {
		panic(fmt.Sprintf("assertion failed: "+msg, v...))
	}
}

func TestNewService(t *testing.T) {
	var foo Foo
	s := newService(&foo)
	fmt.Println(len(s.method))
	_assert(len(s.method) == 1, "wrong service Method, expect 1, but got %d", len(s.method))
	metType := s.method["Sum"]
	fmt.Println(metType)
	_assert(metType != nil, "wrong Method, Sum shouldn't nil")
}

func TestMethodType_Call(t *testing.T) {
	var foo Foo
	s := newService(&foo)
	metType := s.method["Sum"]

	argv := metType.newArgv()
	replyv := metType.newReplyv()
	argv.Set(reflect.ValueOf(Args{Num1: 1, Num2: 2}))
	err := s.call(metType, argv, replyv)
	fmt.Println(err)
	fmt.Println(metType.numCalls)
	fmt.Println(*replyv.Interface().(*int))
	_assert(err == nil && *replyv.Interface().(*int) == 3 && metType.NumCalls() == 1, "failed to call Foo.Sum")
}
