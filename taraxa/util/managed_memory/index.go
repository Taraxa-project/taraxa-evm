package managed_memory

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"reflect"
	"strings"
	"sync"
)

// TODO get rid of this

const addr_prefix = "__ptr__"

type Address = string
type Functions map[string]interface{}
type memCell struct {
	objRef          reflect.Value
	destructor      func()
	destructionLock sync.Mutex
}

type ManagedMemory struct {
	Functions
	mem sync.Map
}

func (this *ManagedMemory) Call(receiverAddr, funcName, argsEncoded string) (retEncoded string, err error) {
	decoder := json.NewDecoder(strings.NewReader(argsEncoded))
	_, err = decoder.Token()
	if err != nil {
		return
	}
	var receiver, callee reflect.Value
	if len(receiverAddr) == 0 {
		receiver, callee = reflect.ValueOf(this), reflect.ValueOf(this.Functions[funcName])
	} else {
		receiver = this.load(receiverAddr).objRef
		method, found := receiver.Type().MethodByName(funcName)
		if !found {
			err = errors.New("Method not found")
			return
		}
		callee = method.Func
	}
	argValues := []reflect.Value{receiver}
	calleeType := callee.Type()
	for i := len(argValues); i < calleeType.NumIn(); i++ {
		argType := calleeType.In(i)
		isPtrArg := argType.Kind() == reflect.Ptr
		if isPtrArg {
			argType = argType.Elem()
		}
		valPtr := reflect.New(argType)
		iValPtr := valPtr.Interface()
		err = decoder.Decode(iValPtr)
		if err != nil {
			return
		}
		if strPtr, castOk := iValPtr.(*string); castOk {
			str := *strPtr
			if strings.HasPrefix(str, addr_prefix) {
				valPtr = this.load(str).objRef
			}
		}
		val := valPtr
		if !isPtrArg {
			val = val.Elem()
		}
		argValues = append(argValues, val)
	}
	var resultValuesInterfaces []interface{}
	resultValues := callee.Call(argValues)
	for _, val := range resultValues {
		resultValuesInterfaces = append(resultValuesInterfaces, val.Interface())
	}
	ret, marshalErr := json.Marshal(resultValuesInterfaces)
	return string(ret), marshalErr
}

func (this *ManagedMemory) Alloc(objPtr interface{}, destructor func()) (addr Address, err error) {
	val := reflect.ValueOf(objPtr)
	if val.Kind() != reflect.Ptr {
		err = errors.New("Only pointers are supported")
		return
	}
	cell := new(memCell)
	cell.objRef = val
	cell.destructor = destructor
	for i := 0; i < 10; i++ {
		addr = addr_prefix + uuid.New().String()
		if _, hasBeenAllocated := this.mem.LoadOrStore(addr, cell); !hasBeenAllocated {
			return
		}
	}
	err = errors.New("Too many attempts")
	return
}

func (this *ManagedMemory) Free(addr Address) error {
	cell := this.load(addr)
	if cell == nil {
		return errors.New("Dangling pointer")
	}
	cell.destructionLock.Lock()
	defer cell.destructionLock.Unlock()
	cell = this.load(addr)
	if cell != nil {
		this.mem.Delete(addr)
		if cell.destructor != nil {
			cell.destructor()
		}
	}
	return nil
}

func (this *ManagedMemory) load(addr Address) *memCell {
	cell, _ := this.mem.Load(addr)
	return cell.(*memCell)
}
