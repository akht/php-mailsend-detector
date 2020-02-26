package visitor

import (
	"strings"

	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/expr/assign"
	"github.com/z7zmey/php-parser/node/name"
	"github.com/z7zmey/php-parser/node/stmt"

	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/visitor"
	"github.com/z7zmey/php-parser/walker"
)

// Dumper writes ast hierarchy to an io.Writer
// Also prints comments and positions attached to nodes
type Visitor struct {
	// Writer     io.Writer
	Indent        string
	NsResolver    *visitor.NamespaceResolver
	DefineNodes   []*expr.FunctionCall
	AssignNodes   []*assign.Assign
	FunctionNodes []*stmt.Function
	MailArguments []string
}

// EnterNode is invoked at every node in hierarchy
func (v *Visitor) EnterNode(w walker.Walkable) bool {
	n := w.(node.Node)

	const targetFunctionName = "mb_send_mail"

	switch n := n.(type) {
	case *assign.Assign:
		v.AssignNodes = append(v.AssignNodes, n)

	case *expr.FunctionCall:
		functionName := funcName(n)
		if functionName == "define" {
			v.DefineNodes = append(v.DefineNodes, n)
		}
		if functionName != targetFunctionName {
			return true
		}

		v.MailArguments = argVarNames(n)

	case *stmt.Function:
		v.FunctionNodes = append(v.FunctionNodes, n)
	}

	return true
}

// LeaveNode is invoked after node process
func (v *Visitor) LeaveNode(n walker.Walkable) {
	// do nothing
}

// GetChildrenVisitor is invoked at every node parameter that contains children nodes
func (v *Visitor) EnterChildNode(key string, w walker.Walkable) {
	// fmt.Fprintf(v.Writer, "%v%q:\n", v.Indent+"  ", key)
	v.Indent = v.Indent + "    "
}

func (v *Visitor) LeaveChildNode(key string, w walker.Walkable) {
	v.Indent = strings.TrimSuffix(v.Indent, "    ")
}

func (v *Visitor) EnterChildList(key string, w walker.Walkable) {
	// fmt.Fprintf(v.Writer, "%v%q:\n", v.Indent+"  ", key)
	v.Indent = v.Indent + "    "
}

func (v *Visitor) LeaveChildList(key string, w walker.Walkable) {
	v.Indent = strings.TrimSuffix(v.Indent, "    ")
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
