package detector

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/expr/assign"
	"github.com/z7zmey/php-parser/node/name"
	"github.com/z7zmey/php-parser/node/scalar"
	"github.com/z7zmey/php-parser/node/stmt"
	"github.com/z7zmey/php-parser/php5"
	"github.com/z7zmey/php-parser/visitor"
)

type Detector struct {
	src io.Reader
}

func NewDetector(src io.Reader) *Detector {
	return &Detector{src: src}
}

func (d *Detector) Detect() string {
	return inspectMailSend(d.src)
}

func (d *Detector) DumpAst() {
	parser := php5.NewParser(d.src, "example.php")
	parser.Parse()

	for _, e := range parser.GetErrors() {
		fmt.Println(e)
	}

	visitor := visitor.PrettyJsonDumper{
		Writer: os.Stdout,
	}

	rootNode := parser.GetRootNode()

	rootNode.Walk(&visitor)
}

func inspectMailSend(src io.Reader) string {
	parser := php5.NewParser(src, "example.php")
	parser.Parse()

	for _, e := range parser.GetErrors() {
		fmt.Println(e)
	}

	rootNode := parser.GetRootNode()

	const targetFunctionName = "mb_send_mail"

	root := rootNode.(*node.Root)

	var assignNodes []*assign.Assign
	var arguments []string

	for _, stmtNode := range root.Stmts {
		expressionNode, ok := stmtNode.(*stmt.Expression)
		if !ok {
			continue
		}

		switch expressionNode.Expr.(type) {
		case *assign.Assign:
			node := expressionNode.Expr.(*assign.Assign)
			assignNodes = append(assignNodes, node)

		case *expr.FunctionCall:
			node := expressionNode.Expr.(*expr.FunctionCall)

			functionName := funcName(node)
			if functionName != targetFunctionName {
				continue
			}

			arguments = argVarNames(node)
		}
	}

	// mail := findVariableValue(assignNodes, arguments)
	mail := make(map[string]string, 3)
	for i, arg := range arguments {
		switch i {
		case 0:
			mail["to"] = findVariableValue(assignNodes, arg)
		case 1:
			mail["subject"] = findVariableValue(assignNodes, arg)
		case 2:
			mail["body"] = findVariableValue(assignNodes, arg)
		}
	}

	var out bytes.Buffer
	out.WriteString("[件名]:" + "\n")
	out.WriteString(mail["subject"] + "\n")
	out.WriteString("[本文]:" + "\n")
	out.WriteString(mail["body"])

	return out.String()
}

// 変数に割り当てられている値を返す
func findVariableValue(nodes []*assign.Assign, name string) string {
	ret := ""

	for _, assignNode := range nodes {
		variableNode := assignNode.Variable.(*expr.Variable)
		variableName := variableNode.VarName.(*node.Identifier).Value
		if variableName != name {
			continue
		}

		switch assignNode.Expression.(type) {
		case *scalar.String:
			stringNode := assignNode.Expression.(*scalar.String)
			value := stringNode.Value

			if len(value) > 0 && value[0] == '"' {
				value = value[1:]
			}
			if len(value) > 0 && value[len(value)-1] == '"' {
				value = value[:len(value)-1]
			}
			ret = value
		case *expr.Variable:
			variableNode := assignNode.Expression.(*expr.Variable)
			varName := variableNode.VarName.(*node.Identifier).Value
			ret = findVariableValue(nodes, varName)
		}
	}

	return ret
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

	funcName := ""
	for _, part := range functionNode.Parts {
		namePartNode, ok := part.(*name.NamePart)
		if !ok {
			continue
		}

		if namePartNode.Value != "" {
			funcName = namePartNode.Value
			break
		}
	}

	return funcName
}

// 引数として渡されている変数名を配列で返す
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
