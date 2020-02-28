package detector

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/akht/php-mailsend-detector/object"
	myvisitor "github.com/akht/php-mailsend-detector/visitor"

	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/expr/assign"
	"github.com/z7zmey/php-parser/node/expr/binary"
	"github.com/z7zmey/php-parser/node/name"
	"github.com/z7zmey/php-parser/node/scalar"
	"github.com/z7zmey/php-parser/node/stmt"
	"github.com/z7zmey/php-parser/php5"
	"github.com/z7zmey/php-parser/visitor"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

type Detector struct {
	src     io.Reader
	parser  *php5.Parser
	visitor myvisitor.Visitor
	env     map[string]object.Object
}

func NewDetector(src io.Reader) *Detector {
	return &Detector{
		src:     src,
		visitor: myvisitor.Visitor{},
		env:     make(map[string]object.Object),
	}
}

func (d *Detector) Detect() string {
	return d.inspectMailSend()
}

// ASTをダンプ
func (d *Detector) DumpAst() {
	parser := php5.NewParser(d.src, "example.php")
	parser.Parse()
	d.parser = parser

	for _, e := range parser.GetErrors() {
		fmt.Println(e)
	}

	visitor := visitor.PrettyJsonDumper{
		Writer: os.Stdout,
	}

	rootNode := parser.GetRootNode()

	rootNode.Walk(&visitor)
}

func (d *Detector) inspectMailSend() string {
	parser := php5.NewParser(d.src, "example.php")
	parser.Parse()
	d.parser = parser

	for _, e := range parser.GetErrors() {
		fmt.Println(e)
	}

	rootNode := parser.GetRootNode()
	rootNode.Walk(&d.visitor)

	d.eval(rootNode)

	mail := make(map[string]string, 3)
	for i, arg := range d.visitor.MailArguments {
		switch i {
		case 0:
			mail["to"] = d.env[arg].(*object.String).Value
		case 1:
			mail["subject"] = d.env[arg].(*object.String).Value
		case 2:
			mail["body"] = d.env[arg].(*object.String).Value
		}
	}

	var out bytes.Buffer
	out.WriteString("[件名]:" + "\n")
	out.WriteString(mail["subject"] + "\n")
	out.WriteString("[本文]:" + "\n")
	out.WriteString(mail["body"])

	return out.String()
}

// パースした全てのノードを返す
func (d *Detector) allNode() []node.Node {
	rootNode := d.parser.GetRootNode()
	root := rootNode.(*node.Root)

	return root.Stmts
}

func (d *Detector) eval(n node.Node) object.Object {
	switch n := n.(type) {
	case *node.Root:
		return d.evalProgram(n)

	case *stmt.Expression:
		return d.eval(n.Expr)

	case *assign.Assign:
		variableName := n.Variable.(*expr.Variable).VarName.(*node.Identifier).Value
		obj := d.eval(n.Expression)
		d.env[variableName] = obj

	case *node.Identifier:
		return d.evalIdentifier(n)

	case *expr.ConstFetch:
		return d.findConstantValue(n)

	case *binary.Concat:
		left := d.eval(n.Left)
		right := d.eval(n.Right)
		return d.evalInfixExpression(".", left, right)

	case *scalar.String:
		value := unwrapString(n.Value)
		return &object.String{Value: value}

	case *expr.Variable:
		return d.eval(n.VarName)

	case *stmt.If:
		return d.evalIf(n)

	case *binary.Equal:
		left := d.eval(n.Left)
		right := d.eval(n.Right)
		return d.evalInfixExpression("==", left, right)

	case *expr.FunctionCall:
		functionName := funcName(n)

		for _, functionNode := range d.visitor.FunctionNodes {
			if functionName == functionNode.FunctionName.(*node.Identifier).Value {
				return d.eval(functionNode)
			}
		}

		functionNode := d.findFunctionDefenition(functionName)
		return d.eval(functionNode)

	case *stmt.Function:
		for _, stmtNode := range n.Stmts {
			statement := d.eval(stmtNode)

			if _, ok := stmtNode.(*stmt.Return); ok {
				return statement
			}
		}

	case *stmt.Return:
		return d.eval(n.Expr)
	}

	return NULL
}

func (d *Detector) evalProgram(root *node.Root) object.Object {
	var result object.Object

	for _, statement := range root.Stmts {
		result = d.eval(statement)
	}

	return result
}

func (d *Detector) evalIdentifier(ident *node.Identifier) object.Object {
	if obj, ok := d.env[ident.Value]; ok {
		return obj
	}

	return NULL
}

func (d *Detector) evalIf(ifNode *stmt.If) object.Object {
	condition := d.eval(ifNode.Cond)

	if condition == TRUE {
		stmts := ifNode.Stmt.(*stmt.StmtList).Stmts
		for i, stmtNode := range stmts {
			if i == len(stmts)-1 {
				return d.eval(stmtNode)
			}
			d.eval(stmtNode)
		}
	}

	stmts := ifNode.Else.(*stmt.Else).Stmt.(*stmt.StmtList).Stmts
	for i, stmtNode := range stmts {
		if i == len(stmts)-1 {
			return d.eval(stmtNode)
		}
		d.eval(stmtNode)
	}

	return NULL
}

func (d *Detector) evalInfixExpression(operator string, left, right object.Object) object.Object {
	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)

	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)

	case operator == "==":
		return nativeBoolToBooleanObject(left == right)

	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)

	default:
		return NULL
	}
}

func evalIntegerInfixExpression(operator string, left, right object.Object) object.Object {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value

	switch operator {
	case "+":
		return &object.Integer{Value: leftVal + rightVal}

	case "-":
		return &object.Integer{Value: leftVal - rightVal}

	case "*":
		return &object.Integer{Value: leftVal * rightVal}

	case "/":
		return &object.Integer{Value: leftVal / rightVal}

	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)

	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)

	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)

	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)

	default:
		return NULL
	}
}

func evalStringInfixExpression(operator string, left, right object.Object) object.Object {
	leftVal := left.(*object.String).Value
	rightVal := right.(*object.String).Value

	switch operator {
	case ".":
		return &object.String{Value: leftVal + rightVal}

	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)

	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)

	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)

	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)

	default:
		return NULL
	}
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}

	return FALSE
}

// FunctionCallノードで呼び出してるFunctionノードを探して返す
func (d *Detector) findFunctionDefenition(functionName string) *stmt.Function {
	for _, functionNode := range d.visitor.FunctionNodes {
		if functionName == functionNode.FunctionName.(*node.Identifier).Value {
			return functionNode
		}
	}

	return &stmt.Function{}
}

// 変数に割り当てられている値を返す
func (d *Detector) findVariableValue(name string) object.Object {
	for _, assignNode := range d.visitor.AssignNodes {
		variableNode := assignNode.Variable.(*expr.Variable)
		variableName := variableNode.VarName.(*node.Identifier).Value
		if variableName != name {
			continue
		}

		return d.eval(assignNode)
	}

	return NULL
}

// 定数に割り当てられている値を返す
func (d *Detector) findConstantValue(constFecthNode *expr.ConstFetch) object.Object {

	nameNode := constFecthNode.Constant.(*name.Name)
	partsNode := nameNode.Parts[0].(*name.NamePart)
	constantName := partsNode.Value

	for _, funcCallNode := range d.visitor.DefineNodes {
		argumentList := funcCallNode.ArgumentList
		argument := argumentList.Arguments[0].(*node.Argument)
		definedConstantName := d.eval(argument.Expr)

		stringObj, ok := definedConstantName.(*object.String)
		if !ok {
			continue
		}

		if stringObj.Value != constantName {
			continue
		}

		argument = argumentList.Arguments[1].(*node.Argument)
		return d.eval(argument.Expr)
	}

	return NULL
}

// 関数呼び出しを復元する
func funcCallExprStr(funcName string, argVarNames []string) string {
	argumentStr := ""
	for i, arg := range argVarNames {
		if i > 0 {
			argumentStr += ", "
		}
		argumentStr += "$" + arg
	}

	return funcName + "(" + argumentStr + ");"
}

// 関数名を返す
func funcName(node *expr.FunctionCall) string {
	functionNode := node.Function.(*name.Name)

	for _, part := range functionNode.Parts {
		namePartNode, ok := part.(*name.NamePart)
		if !ok {
			continue
		}

		if namePartNode.Value != "" {
			return namePartNode.Value
		}
	}

	return ""
}

// 関数呼び出しの引数として渡されている変数名を配列で返す
func argVarNames(functionCallNode *expr.FunctionCall) []string {
	var ret []string

	argumentList := functionCallNode.ArgumentList
	for _, argNode := range argumentList.Arguments {
		argument := argNode.(*node.Argument)

		variable, ok := argument.Expr.(*expr.Variable)
		if !ok {
			continue
		}

		variableName := variable.VarName.(*node.Identifier).Value
		ret = append(ret, variableName)
	}

	return ret
}

func unwrapString(value string) string {
	if len(value) > 0 && (value[0] == '"' || value[0] == '\'') {
		value = value[1:]
	}
	if len(value) > 0 && (value[len(value)-1] == '"' || value[len(value)-1] == '\'') {
		value = value[:len(value)-1]
	}

	return value
}
