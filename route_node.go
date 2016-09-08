package wemvc

import (
	"errors"
	"fmt"
	"strings"
)

type pathType uint8

const (
	static pathType = iota
	root
	param
	catchAll

	paramBegin = '<'
	paramBeginStr = "<"
	paramEnd  = '>'
	paramEndStr = ">"
	pathInfo = "*pathInfo"
)

type RouteOption struct {
	Validation string
	Setting    string
	MaxLength  uint8
	MinLength  uint8
}

type routeNode struct {
	NodeType  pathType
	CurDepth  uint16
	MaxDepth  uint16
	Path      string
	ParamPath string
	Params    map[string]RouteOption
	CtrlInfo  *controllerInfo
	Children  []*routeNode
}

func (node *routeNode) isLeaf() bool {
	if node.NodeType == root {
		return false
	}
	return node.hasChildren() == false
}

func (node *routeNode) hasChildren() bool {
	return len(node.Children) > 0
}

func (node *routeNode) findChild(path string) *routeNode {
	if !node.hasChildren() {
		return nil
	}
	for _, child := range node.Children {
		if child.Path == path {
			return child
		}
	}
	return nil
}

func (node *routeNode) addChild(childNode *routeNode) error {
	if childNode == nil {
		return errors.New("'childNode' parameter cannot be nil")
	}
	var existChild = node.findChild(childNode.Path)
	if existChild == nil {
		node.Children = append(node.Children, childNode)
		return nil
	}
	if childNode.MaxDepth > existChild.MaxDepth {
		existChild.MaxDepth = childNode.MaxDepth
	}
	if childNode.isLeaf() {
		if existChild.CtrlInfo != nil {
			return errors.New(fmt.Sprintf("Duplicate controller info in route tree. Path: %s, Depth: %d",
				existChild.Path,
				existChild.CurDepth))
		}
		existChild.CtrlInfo = childNode.CtrlInfo
	} else {
		for _, child := range childNode.Children {
			err := existChild.addChild(child)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func newRouteNode(routePath string, ctrlInfo *controllerInfo) (*routeNode, error) {
	err := checkRoutePath(routePath)
	if err != nil {
		return nil, err
	}
	splitPaths, err := splitUrlPath(routePath)
	if err != nil {
		return nil, err
	}
	var length = uint16(len(splitPaths))
	if length == 0 {
		return nil, nil
	}
	if detectNodeType(splitPaths[length-1]) == catchAll {
		length = 255
	}
	var result *routeNode
	var current *routeNode
	for i, p := range splitPaths {
		var child = &routeNode{
			NodeType: detectNodeType(p),
			CurDepth: uint16(i + 1),
			MaxDepth: uint16(length - uint16(i)),
			Path:     p,
		}
		if child.NodeType == param {
			paramPath, params, err := analyzeParamOption(child.Path)
			if err != nil {
				return nil, err
			} else {
				child.ParamPath = paramPath
				child.Params = params
			}
		}
		if result == nil {
			result = child
			current = result
		} else {
			current.Children = []*routeNode{child}
			current = current.Children[0]
		}
	}
	current.CtrlInfo = ctrlInfo
	current = result
	for {
		if current == nil {
			break
		}
		if strings.Contains(current.Path, "*") && current.NodeType != catchAll {
			return nil, errors.New("Invalid URL route parameter '" + current.Path + "'")
		}
		if current.NodeType == catchAll && len(current.Children) > 0 {
			return nil, errors.New("Invalid route'" + routePath + ". " +
				"The '*pathInfo' parameter should be at the end of the route. " +
				"For example: '/shell/*pathInfo'.")
		}
		if len(current.Children) > 0 {
			current = current.Children[0]
		} else {
			current = nil
		}
	}
	return result, nil
}