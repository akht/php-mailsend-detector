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
	src         io.Reader
	parser      *php5.Parser
	assignNodes []*assign.Assign
}

func NewDetector(src io.Reader) *Detector {
	return &Detector{src: src}
}

func (d *Detector) Detect() string {
	return d.inspectMailSend()
}

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

	const targetFunctionName = "mb_send_mail"

	var assignNodes []*assign.Assign
	var arguments []string

	for _, stmtNode := range d.allNode() {
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

	d.assignNodes = assignNodes

	mail := make(map[string]string, 3)
	for i, arg := range arguments {
		switch i {
		case 0:
			mail["to"] = d.findVariableValue(arg)
		case 1:
			mail["subject"] = d.findVariableValue(arg)
		case 2:
			mail["body"] = d.findVariableValue(arg)
		}
	}

	var out bytes.Buffer
	out.WriteString("[件名]:" + "\n")
	out.WriteString(mail["subject"] + "\n")
	out.WriteString("[本文]:" + "\n")
	out.WriteString(mail["body"])

	return out.String()
}

func (d *Detector) allNode() []node.Node {
	rootNode := d.parser.GetRootNode()
	root := rootNode.(*node.Root)

	return root.Stmts
}

func (d *Detector) eval(n node.Node) string {
	ret := ""

	switch n.(type) {
	case *assign.Assign:
		assignNode := n.(*assign.Assign)
		return d.eval(assignNode.Expression)

	case *scalar.String:
		stringNode := n.(*scalar.String)
		value := stringNode.Value

		if len(value) > 0 && value[0] == '"' {
			value = value[1:]
		}
		if len(value) > 0 && value[len(value)-1] == '"' {
			value = value[:len(value)-1]
		}
		ret = value

	case *expr.Variable:
		variableNode := n.(*expr.Variable)
		varName := variableNode.VarName.(*node.Identifier).Value
		ret = d.findVariableValue(varName)

	case *expr.FunctionCall:
		functionCallNode := n.(*expr.FunctionCall)

		return d.eval(functionCallNode.Function)

	case *stmt.Function:
		ret = "Function"
	}

	return ret
}

// 変数に割り当てられている値を返す
func (d *Detector) findVariableValue(name string) string {
	ret := ""

	for _, assignNode := range d.assignNodes {
		variableNode := assignNode.Variable.(*expr.Variable)
		variableName := variableNode.VarName.(*node.Identifier).Value
		if variableName != name {
			continue
		}

		ret = d.eval(assignNode)
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
