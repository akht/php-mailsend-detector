package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/expr/assign"
	"github.com/z7zmey/php-parser/node/name"
	"github.com/z7zmey/php-parser/node/scalar"
	"github.com/z7zmey/php-parser/node/stmt"
	"github.com/z7zmey/php-parser/php5"
)

type Mail struct {
	To      string
	From    string
	Subject string
	Body    string
}

func main() {
	// ファイルをOpenする
	f, err := os.Open("test.php")
	if err != nil {
		fmt.Println("error")
	}
	defer f.Close()

	// 一気に全部読み取り
	b, err := ioutil.ReadAll(f)
	src := bytes.NewBufferString(string(b))

	// src := bytes.NewBufferString(`<? echo "Hello world";`)

	parser := php5.NewParser(src, "example.php")
	parser.Parse()

	for _, e := range parser.GetErrors() {
		fmt.Println(e)
	}

	// visitor := visitor.Dumper{
	// 	Writer: os.Stdout,
	// 	Indent: "",
	// 	// Memo:   make(map[string]int),
	// }

	// var nsResolver *visitor.NamespaceResolver
	// visitor := visitor.PrettyJsonDumper{
	// 	Writer: os.Stdout,
	// 	// NsResolver: nsResolver,
	// }

	rootNode := parser.GetRootNode()

	// rootNode.Walk(&visitor)

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

	// 関数呼び出しの文字列を復元
	// funcCallExprStr := funcCallExprStr(targetFunctionName, arguments)
	// fmt.Println(funcCallExprStr)

	mail := findVariableValue(assignNodes, arguments)
	fmt.Println("[件名]:")
	fmt.Println(mail["subject"])
	fmt.Println("[本文]:")
	fmt.Println(mail["body"])
}

// 変数に割り当てられている値を返す
func findVariableValue(nodes []*assign.Assign, names []string) map[string]string {
	keys := []string{"to", "subject", "body"}
	ret := make(map[string]string, len(keys))

	for i, name := range names {
		for _, assignNode := range nodes {
			variableNode := assignNode.Variable.(*expr.Variable)
			variableName := variableNode.VarName.(*node.Identifier).Value
			if variableName != name {
				continue
			}
			value := assignNode.Expression.(*scalar.String).Value
			if i < len(keys) {
				ret[keys[i]] = value
			}
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
