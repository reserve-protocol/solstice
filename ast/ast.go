package ast

import (
    "strconv"
    "strings"

    "github.com/coordination-institute/debugging-tools/solc"
    "github.com/coordination-institute/debugging-tools/srclocation"
)

// The Abstract Syntax Tree of a contract.
type ASTTree struct {
    Id uint
    SrcLoc srclocation.SourceLocation
    Children []*ASTTree
}

func Get(contractName string) (ASTTree, error) {
    srcMapJSON, err := solc.GetCombinedJSON("ast", []string{contractName})
    if err != nil {
        return ASTTree{}, err
    }

    processedTree, err := processASTNode(
        srcMapJSON.Sources[contractName].AST,
        srcMapJSON.SourceList,
    )

    return processedTree, err
}

func processASTNode(node solc.JSONASTTree, sourceList []string) (ASTTree, error) {
    var newTree ASTTree
    newTree.Id = node.Id

    srcLocParts := strings.Split(node.Src, ":")

    byteOffset, err := strconv.Atoi(srcLocParts[0])
    if err != nil {
        return newTree, err
    }

    byteLength, err := strconv.Atoi(srcLocParts[1])
    if err != nil {
        return newTree, err
    }

    sourceFileIndex, err := strconv.Atoi(srcLocParts[2])
    if err != nil {
        return newTree, err
    }

    newTree.SrcLoc = srclocation.SourceLocation{
        byteOffset,
        byteLength,
        sourceList[sourceFileIndex],
        *new(rune),
    }

    for _, childNode := range node.Children {
        newNode, err := processASTNode(*childNode, sourceList)
        if err != nil {
            return newTree, err
        }
        newTree.Children = append(newTree.Children, &newNode)
    }

    return newTree, nil
}
