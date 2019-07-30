package compiler

// This file transforms interface-related instructions (*ssa.MakeInterface,
// *ssa.TypeAssert, calls on interface types) to an intermediate IR form, to be
// lowered to the final form by the interface lowering pass. See
// interface-lowering.go for more details.

import (
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/tinygo-org/tinygo/ir"
	"golang.org/x/tools/go/ssa"
	"tinygo.org/x/go-llvm"
)

// parseMakeInterface emits the LLVM IR for the *ssa.MakeInterface instruction.
// It tries to put the type in the interface value, but if that's not possible,
// it will do an allocation of the right size and put that in the interface
// value field.
//
// An interface value is a {typecode, value} tuple, or {i16, i8*} to be exact.
func (c *Compiler) parseMakeInterface(val llvm.Value, typ types.Type, pos token.Pos) llvm.Value {
	itfValue := c.emitPointerPack([]llvm.Value{val})
	itfTypeCodeGlobal := c.getTypeCode(typ)
	itfMethodSetGlobal := c.getTypeMethodSet(typ)
	itfConcreteTypeGlobal := c.mod.NamedGlobal("typeInInterface:" + itfTypeCodeGlobal.Name())
	if itfConcreteTypeGlobal.IsNil() {
		typeInInterface := c.getLLVMRuntimeType("typeInInterface")
		itfConcreteTypeGlobal = llvm.AddGlobal(c.mod, typeInInterface, "typeInInterface:"+itfTypeCodeGlobal.Name())
		itfConcreteTypeGlobal.SetInitializer(llvm.ConstNamedStruct(typeInInterface, []llvm.Value{itfTypeCodeGlobal, itfMethodSetGlobal}))
		itfConcreteTypeGlobal.SetGlobalConstant(true)
		itfConcreteTypeGlobal.SetLinkage(llvm.PrivateLinkage)
	}
	itfTypeCode := c.builder.CreatePtrToInt(itfConcreteTypeGlobal, c.uintptrType, "")
	itf := llvm.Undef(c.getLLVMRuntimeType("_interface"))
	itf = c.builder.CreateInsertValue(itf, itfTypeCode, 0, "")
	itf = c.builder.CreateInsertValue(itf, itfValue, 1, "")
	return itf
}

// getTypeCode returns a reference to a type code.
// It returns a pointer to an external global which should be replaced with the
// real type in the interface lowering pass.
func (c *Compiler) getTypeCode(typ types.Type) llvm.Value {
	globalName := "type:" + getTypeCodeName(typ)
	global := c.mod.NamedGlobal(globalName)
	if global.IsNil() {
		global = llvm.AddGlobal(c.mod, c.getLLVMRuntimeType("typecodeID"), globalName)
		global.SetGlobalConstant(true)
	}
	return global
}

// getTypeCodeName returns a name for this type that can be used in the
// interface lowering pass to assign type codes as expected by the reflect
// package. See getTypeCodeNum.
func getTypeCodeName(t types.Type) string {
	name := ""
	if named, ok := t.(*types.Named); ok {
		name = "~" + named.String() + ":"
		t = t.Underlying()
	}
	switch t := t.(type) {
	case *types.Array:
		return "array:" + name + strconv.FormatInt(t.Len(), 10) + ":" + getTypeCodeName(t.Elem())
	case *types.Basic:
		var kind string
		switch t.Kind() {
		case types.Bool:
			kind = "bool"
		case types.Int:
			kind = "int"
		case types.Int8:
			kind = "int8"
		case types.Int16:
			kind = "int16"
		case types.Int32:
			kind = "int32"
		case types.Int64:
			kind = "int64"
		case types.Uint:
			kind = "uint"
		case types.Uint8:
			kind = "uint8"
		case types.Uint16:
			kind = "uint16"
		case types.Uint32:
			kind = "uint32"
		case types.Uint64:
			kind = "uint64"
		case types.Uintptr:
			kind = "uintptr"
		case types.Float32:
			kind = "float32"
		case types.Float64:
			kind = "float64"
		case types.Complex64:
			kind = "complex64"
		case types.Complex128:
			kind = "complex128"
		case types.String:
			kind = "string"
		case types.UnsafePointer:
			kind = "unsafeptr"
		default:
			panic("unknown basic type: " + t.Name())
		}
		return "basic:" + name + kind
	case *types.Chan:
		return "chan:" + name + getTypeCodeName(t.Elem())
	case *types.Interface:
		methods := make([]string, t.NumMethods())
		for i := 0; i < t.NumMethods(); i++ {
			methods[i] = getTypeCodeName(t.Method(i).Type())
		}
		return "interface:" + name + "{" + strings.Join(methods, ",") + "}"
	case *types.Map:
		keyType := getTypeCodeName(t.Key())
		elemType := getTypeCodeName(t.Elem())
		return "map:" + name + "{" + keyType + "," + elemType + "}"
	case *types.Pointer:
		return "pointer:" + name + getTypeCodeName(t.Elem())
	case *types.Signature:
		params := make([]string, t.Params().Len())
		for i := 0; i < t.Params().Len(); i++ {
			params[i] = getTypeCodeName(t.Params().At(i).Type())
		}
		results := make([]string, t.Results().Len())
		for i := 0; i < t.Results().Len(); i++ {
			results[i] = getTypeCodeName(t.Results().At(i).Type())
		}
		return "func:" + name + "{" + strings.Join(params, ",") + "}{" + strings.Join(results, ",") + "}"
	case *types.Slice:
		return "slice:" + name + getTypeCodeName(t.Elem())
	case *types.Struct:
		elems := make([]string, t.NumFields())
		if t.NumFields() > 2 && t.Field(0).Name() == "C union" {
			// TODO: report this as a normal error instead of panicking.
			panic("cgo unions are not allowed in interfaces")
		}
		for i := 0; i < t.NumFields(); i++ {
			elems[i] = getTypeCodeName(t.Field(i).Type())
		}
		return "struct:" + name + "{" + strings.Join(elems, ",") + "}"
	default:
		panic("unknown type: " + t.String())
	}
}

// getTypeMethodSet returns a reference (GEP) to a global method set. This
// method set should be unreferenced after the interface lowering pass.
func (c *Compiler) getTypeMethodSet(typ types.Type) llvm.Value {
	global := c.mod.NamedGlobal(typ.String() + "$methodset")
	zero := llvm.ConstInt(c.ctx.Int32Type(), 0, false)
	if !global.IsNil() {
		// the method set already exists
		return llvm.ConstGEP(global, []llvm.Value{zero, zero})
	}

	ms := c.ir.Program.MethodSets.MethodSet(typ)
	if ms.Len() == 0 {
		// no methods, so can leave that one out
		return llvm.ConstPointerNull(llvm.PointerType(c.getLLVMRuntimeType("interfaceMethodInfo"), 0))
	}

	methods := make([]llvm.Value, ms.Len())
	interfaceMethodInfoType := c.getLLVMRuntimeType("interfaceMethodInfo")
	for i := 0; i < ms.Len(); i++ {
		method := ms.At(i)
		signatureGlobal := c.getMethodSignature(method.Obj().(*types.Func))
		f := c.ir.Program.MethodValue(method)
		if c.getFunction(f).IsNil() {
			// compiler error, so panic
			panic("cannot find function: " + c.getFunctionInfo(f).linkName)
		}
		fn := c.getInterfaceInvokeWrapper(f)
		methodInfo := llvm.ConstNamedStruct(interfaceMethodInfoType, []llvm.Value{
			signatureGlobal,
			llvm.ConstPtrToInt(fn, c.uintptrType),
		})
		methods[i] = methodInfo
	}
	arrayType := llvm.ArrayType(interfaceMethodInfoType, len(methods))
	value := llvm.ConstArray(interfaceMethodInfoType, methods)
	global = llvm.AddGlobal(c.mod, arrayType, typ.String()+"$methodset")
	global.SetInitializer(value)
	global.SetGlobalConstant(true)
	global.SetLinkage(llvm.PrivateLinkage)
	return llvm.ConstGEP(global, []llvm.Value{zero, zero})
}

// getInterfaceMethodSet returns a global variable with the method set of the
// given named interface type. This method set is used by the interface lowering
// pass.
func (c *Compiler) getInterfaceMethodSet(typ *types.Named) llvm.Value {
	global := c.mod.NamedGlobal(typ.String() + "$interface")
	zero := llvm.ConstInt(c.ctx.Int32Type(), 0, false)
	if !global.IsNil() {
		// method set already exist, return it
		return llvm.ConstGEP(global, []llvm.Value{zero, zero})
	}

	// Every method is a *i16 reference indicating the signature of this method.
	methods := make([]llvm.Value, typ.Underlying().(*types.Interface).NumMethods())
	for i := range methods {
		method := typ.Underlying().(*types.Interface).Method(i)
		methods[i] = c.getMethodSignature(method)
	}

	value := llvm.ConstArray(methods[0].Type(), methods)
	global = llvm.AddGlobal(c.mod, value.Type(), typ.String()+"$interface")
	global.SetInitializer(value)
	global.SetGlobalConstant(true)
	global.SetLinkage(llvm.PrivateLinkage)
	return llvm.ConstGEP(global, []llvm.Value{zero, zero})
}

// getMethodSignature returns a global variable which is a reference to an
// external *i16 indicating the indicating the signature of this method. It is
// used during the interface lowering pass.
func (c *Compiler) getMethodSignature(method *types.Func) llvm.Value {
	signature := ir.MethodSignature(method)
	signatureGlobal := c.mod.NamedGlobal("func " + signature)
	if signatureGlobal.IsNil() {
		signatureGlobal = llvm.AddGlobal(c.mod, c.ctx.Int8Type(), "func "+signature)
		signatureGlobal.SetGlobalConstant(true)
	}
	return signatureGlobal
}

// parseTypeAssert will emit the code for a typeassert, used in if statements
// and in type switches (Go SSA does not have type switches, only if/else
// chains). Note that even though the Go SSA does not contain type switches,
// LLVM will recognize the pattern and make it a real switch in many cases.
//
// Type asserts on concrete types are trivial: just compare type numbers. Type
// asserts on interfaces are more difficult, see the comments in the function.
func (c *Compiler) parseTypeAssert(frame *Frame, expr *ssa.TypeAssert) llvm.Value {
	itf := c.getValue(frame, expr.X)
	assertedType := c.getLLVMType(expr.AssertedType)

	actualTypeNum := c.builder.CreateExtractValue(itf, 0, "interface.type")
	commaOk := llvm.Value{}
	if _, ok := expr.AssertedType.Underlying().(*types.Interface); ok {
		// Type assert on interface type.
		// This pseudo call will be lowered in the interface lowering pass to a
		// real call which checks whether the provided typecode is any of the
		// concrete types that implements this interface.
		// This is very different from how interface asserts are implemented in
		// the main Go compiler, where the runtime checks whether the type
		// implements each method of the interface. See:
		// https://research.swtch.com/interfaces
		methodSet := c.getInterfaceMethodSet(expr.AssertedType.(*types.Named))
		commaOk = c.createRuntimeCall("interfaceImplements", []llvm.Value{actualTypeNum, methodSet}, "")

	} else {
		// Type assert on concrete type.
		// Call runtime.typeAssert, which will be lowered to a simple icmp or
		// const false in the interface lowering pass.
		assertedTypeCodeGlobal := c.getTypeCode(expr.AssertedType)
		commaOk = c.createRuntimeCall("typeAssert", []llvm.Value{actualTypeNum, assertedTypeCodeGlobal}, "typecode")
	}

	// Add 2 new basic blocks (that should get optimized away): one for the
	// 'ok' case and one for all instructions following this type assert.
	// This is necessary because we need to insert the casted value or the
	// nil value based on whether the assert was successful. Casting before
	// this check tells LLVM that it can use this value and may
	// speculatively dereference pointers before the check. This can lead to
	// a miscompilation resulting in a segfault at runtime.
	// Additionally, this is even required by the Go spec: a failed
	// typeassert should return a zero value, not an incorrectly casted
	// value.

	prevBlock := c.builder.GetInsertBlock()
	okBlock := c.ctx.AddBasicBlock(frame.llvmFn, "typeassert.ok")
	nextBlock := c.ctx.AddBasicBlock(frame.llvmFn, "typeassert.next")
	frame.blockExits[frame.currentBlock] = nextBlock // adjust outgoing block for phi nodes
	c.builder.CreateCondBr(commaOk, okBlock, nextBlock)

	// Retrieve the value from the interface if the type assert was
	// successful.
	c.builder.SetInsertPointAtEnd(okBlock)
	var valueOk llvm.Value
	if _, ok := expr.AssertedType.Underlying().(*types.Interface); ok {
		// Type assert on interface type. Easy: just return the same
		// interface value.
		valueOk = itf
	} else {
		// Type assert on concrete type. Extract the underlying type from
		// the interface (but only after checking it matches).
		valuePtr := c.builder.CreateExtractValue(itf, 1, "typeassert.value.ptr")
		valueOk = c.emitPointerUnpack(valuePtr, []llvm.Type{assertedType})[0]
	}
	c.builder.CreateBr(nextBlock)

	// Continue after the if statement.
	c.builder.SetInsertPointAtEnd(nextBlock)
	phi := c.builder.CreatePHI(assertedType, "typeassert.value")
	phi.AddIncoming([]llvm.Value{c.getZeroValue(assertedType), valueOk}, []llvm.BasicBlock{prevBlock, okBlock})

	if expr.CommaOk {
		tuple := c.ctx.ConstStruct([]llvm.Value{llvm.Undef(assertedType), llvm.Undef(c.ctx.Int1Type())}, false) // create empty tuple
		tuple = c.builder.CreateInsertValue(tuple, phi, 0, "")                                                  // insert value
		tuple = c.builder.CreateInsertValue(tuple, commaOk, 1, "")                                              // insert 'comma ok' boolean
		return tuple
	} else {
		// This is kind of dirty as the branch above becomes mostly useless,
		// but hopefully this gets optimized away.
		c.createRuntimeCall("interfaceTypeAssert", []llvm.Value{commaOk}, "")
		return phi
	}
}

// getInvokeCall creates and returns the function pointer and parameters of an
// interface call. It can be used in a call or defer instruction.
func (c *Compiler) getInvokeCall(frame *Frame, instr *ssa.CallCommon) (llvm.Value, []llvm.Value) {
	// Call an interface method with dynamic dispatch.
	itf := c.getValue(frame, instr.Value) // interface

	llvmFnType := c.getRawFuncType(instr.Method.Type().(*types.Signature))

	typecode := c.builder.CreateExtractValue(itf, 0, "invoke.typecode")
	values := []llvm.Value{
		typecode,
		c.getInterfaceMethodSet(instr.Value.Type().(*types.Named)),
		c.getMethodSignature(instr.Method),
	}
	fn := c.createRuntimeCall("interfaceMethod", values, "invoke.func")
	fnCast := c.builder.CreateIntToPtr(fn, llvmFnType, "invoke.func.cast")
	receiverValue := c.builder.CreateExtractValue(itf, 1, "invoke.func.receiver")

	args := []llvm.Value{receiverValue}
	for _, arg := range instr.Args {
		args = append(args, c.getValue(frame, arg))
	}
	// Add the context parameter. An interface call never takes a context but we
	// have to supply the parameter anyway.
	args = append(args, llvm.Undef(c.i8ptrType))
	// Add the parent goroutine handle.
	args = append(args, llvm.Undef(c.i8ptrType))

	return fnCast, args
}

// interfaceInvokeWrapper keeps some state between getInterfaceInvokeWrapper and
// createInterfaceInvokeWrapper. The former is called during IR construction
// itself and the latter is called when finishing up the IR.
type interfaceInvokeWrapper struct {
	fn           *ssa.Function
	wrapper      llvm.Value
	receiverType llvm.Type
}

// Wrap an interface method function pointer. The wrapper takes in a pointer to
// the underlying value, dereferences it, and calls the real method. This
// wrapper is only needed when the interface value actually doesn't fit in a
// pointer and a pointer to the value must be created.
func (c *Compiler) getInterfaceInvokeWrapper(f *ssa.Function) llvm.Value {
	wrapperName := c.getFunctionInfo(f).linkName + "$invoke"
	wrapper := c.mod.NamedFunction(wrapperName)
	if !wrapper.IsNil() {
		// Wrapper already created. Return it directly.
		return wrapper
	}

	// Get the expanded receiver type.
	receiverType := c.getLLVMType(f.Params[0].Type())
	expandedReceiverType := c.expandFormalParamType(receiverType)

	// Does this method even need any wrapping?
	if len(expandedReceiverType) == 1 && receiverType.TypeKind() == llvm.PointerTypeKind {
		// Nothing to wrap.
		// Casting a function signature to a different signature and calling it
		// with a receiver pointer bitcasted to *i8 (as done in calls on an
		// interface) is hopefully a safe (defined) operation.
		return c.getFunction(f)
	}

	// create wrapper function
	fnType := c.getFunction(f).Type().ElementType()
	paramTypes := append([]llvm.Type{c.i8ptrType}, fnType.ParamTypes()[len(expandedReceiverType):]...)
	wrapFnType := llvm.FunctionType(fnType.ReturnType(), paramTypes, false)
	wrapper = llvm.AddFunction(c.mod, wrapperName, wrapFnType)
	c.interfaceInvokeWrappers = append(c.interfaceInvokeWrappers, interfaceInvokeWrapper{
		fn:           f,
		wrapper:      wrapper,
		receiverType: receiverType,
	})
	return wrapper
}

// createInterfaceInvokeWrapper finishes the work of getInterfaceInvokeWrapper,
// see that function for details.
func (c *Compiler) createInterfaceInvokeWrapper(state interfaceInvokeWrapper) {
	wrapper := state.wrapper
	fn := state.fn
	receiverType := state.receiverType
	wrapper.SetLinkage(llvm.InternalLinkage)
	wrapper.SetUnnamedAddr(true)

	// add debug info if needed
	if c.Debug {
		pos := c.ir.Program.Fset.Position(fn.Pos())
		difunc := c.attachDebugInfoRaw(fn, wrapper, "$invoke", pos.Filename, pos.Line)
		c.builder.SetCurrentDebugLocation(uint(pos.Line), uint(pos.Column), difunc, llvm.Metadata{})
	}

	// set up IR builder
	block := c.ctx.AddBasicBlock(wrapper, "entry")
	c.builder.SetInsertPointAtEnd(block)

	receiverValue := c.emitPointerUnpack(wrapper.Param(0), []llvm.Type{receiverType})[0]
	params := append(c.expandFormalParam(receiverValue), wrapper.Params()[1:]...)
	llvmFn := c.getFunction(fn)
	if llvmFn.Type().ElementType().ReturnType().TypeKind() == llvm.VoidTypeKind {
		c.builder.CreateCall(llvmFn, params, "")
		c.builder.CreateRetVoid()
	} else {
		ret := c.builder.CreateCall(llvmFn, params, "ret")
		c.builder.CreateRet(ret)
	}
}
