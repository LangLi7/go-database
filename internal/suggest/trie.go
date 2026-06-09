package suggest

import (
	"strings"

	"go-database/internal/plugin"
)

type trieNode struct {
	children map[rune]*trieNode
	isEnd    bool
	value    string
}

type Trie struct {
	root *trieNode
}

func NewTrie() *Trie {
	return &Trie{root: &trieNode{children: make(map[rune]*trieNode)}}
}

func (t *Trie) Insert(word string) {
	node := t.root
	for _, ch := range strings.ToLower(word) {
		if node.children[ch] == nil {
			node.children[ch] = &trieNode{children: make(map[rune]*trieNode)}
		}
		node = node.children[ch]
	}
	node.isEnd = true
	node.value = word
}

func (t *Trie) Search(prefix string, limit int) []string {
	node := t.root
	for _, ch := range strings.ToLower(prefix) {
		if node.children[ch] == nil {
			return nil
		}
		node = node.children[ch]
	}
	var results []string
	collect(node, &results, limit)
	return results
}

func collect(node *trieNode, results *[]string, limit int) {
	if len(*results) >= limit {
		return
	}
	if node.isEnd {
		*results = append(*results, node.value)
	}
	for _, child := range node.children {
		collect(child, results, limit)
	}
}

var sqlKeywords = []string{
	"SELECT", "FROM", "WHERE", "INSERT", "INTO", "VALUES", "UPDATE", "SET",
	"DELETE", "CREATE", "TABLE", "DATABASE", "DROP", "ALTER", "ADD", "COLUMN",
	"INDEX", "PRIMARY", "KEY", "FOREIGN", "REFERENCES", "NOT", "NULL",
	"DEFAULT", "UNIQUE", "CHECK", "CONSTRAINT", "JOIN", "INNER", "LEFT",
	"RIGHT", "OUTER", "FULL", "ON", "AND", "OR", "IN", "BETWEEN", "LIKE",
	"ORDER", "BY", "ASC", "DESC", "LIMIT", "OFFSET", "GROUP", "HAVING",
	"COUNT", "SUM", "AVG", "MIN", "MAX", "DISTINCT", "AS", "CASE", "WHEN",
	"THEN", "ELSE", "END", "EXISTS", "ALL", "ANY", "SOME", "UNION", "EXCEPT",
	"INTERSECT", "WITH", "RECURSIVE", "RETURNING", "EXPLAIN", "ANALYZE",
	"VACUUM", "TRUNCATE", "RENAME", "IF", "EXISTS", "CASCADE", "RESTRICT",
	"BEGIN", "COMMIT", "ROLLBACK", "SAVEPOINT", "GRANT", "REVOKE",
	"SHOW", "DESCRIBE", "USE", "PRAGMA", "EXECUTE",
}

func NewKeywordTrie() *Trie {
	t := NewTrie()
	for _, kw := range sqlKeywords {
		t.Insert(kw)
	}
	return t
}

func NewSchemaTrie(schema *plugin.Schema) *Trie {
	t := NewTrie()
	if schema == nil {
		return t
	}
	for _, tbl := range schema.Tables {
		t.Insert(tbl.Name)
		for _, col := range tbl.Columns {
			t.Insert(col.Name)
			t.Insert(tbl.Name + "." + col.Name)
		}
	}
	return t
}
