package myrpc

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

type methodType struct {
	method 	reflect.Method // 方法
	ArgType	reflect.Type	// 第一个参数
	ReplyType	reflect.Type	//第二个参数
	numCalls 	uint64	// 调用次数
}

// 调用次数增加
func (m *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&m.numCalls)
}

// 获取参数类型的实例
func (m *methodType) newArgv() reflect.Value {
	var argv reflect.Value
	if m.ArgType.Kind() == reflect.Ptr {
		// todo
		argv = reflect.New(m.ArgType.Elem())
	} else {
		argv = reflect.New(m.ArgType).Elem()
	}
	return argv
}

// 获取参数类型的实例
func (m *methodType) newReplyv() reflect.Value {
	replyv := reflect.New(m.ReplyType.Elem())
	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	}
	return replyv
}

type service struct {
	name string	// 映射的结构体名称，如WaitGroup
	typ reflect.Type	// 结构体类型
	receiver 	reflect.Value // 结构体实例, 调用方法的时候需要用到
	method map[string]*methodType  // 存储结构体符合条件的方法
}

// 获取service实例
func newService(receiver interface{}) *service {
	s := new(service)
	// 为s赋值
	s.receiver = reflect.ValueOf(receiver)
	// 获取名称。可能是指针类型，需要使用Indirect转换.若是指针类型，且没有使用indirect的话，会得到空字符串
	s.name = reflect.Indirect(s.receiver).Type().Name()
	s.typ = reflect.TypeOf(receiver)
	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.name)
	}
	s.registerMethods()
	return s
}

// 通过反射获该service实例的所有导出方法并复制给该service的method
func (s *service) registerMethods() {
	s.method = make(map[string]*methodType)
	//fmt.Println(s.typ)
	//fmt.Println(s.typ.NumMethod())
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i)
		metType := method.Type
		// 检验参数数量
		if metType.NumIn() != 3 || metType.NumOut() != 1 {
			continue
		}
		//  reflect.TypeOf((*error)(nil)).Elem()的值是error
		if metType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		argType, replyType := metType.In(1), metType.In(2)
		s.method[method.Name] = &methodType{
			method: method,
			ArgType: argType,
			ReplyType: replyType,
		}
		log.Printf("rpc server: register %s.%s\n", s.name, method.Name)
	}
}

// 通过反射执行一个方法
func (s *service) call(metType *methodType, argsType, replyv reflect.Value) error {
	atomic.AddUint64(&metType.numCalls, 1)
	metFunc := metType.method.Func
	returnValues := metFunc.Call([]reflect.Value{s.receiver, argsType, replyv})
	if errInter := returnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	return nil
}
