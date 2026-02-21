//go:build !grammar_blobs_external && grammar_set_core

package grammars

import "embed"

//go:embed grammar_blobs/bash.bin grammar_blobs/c.bin grammar_blobs/cpp.bin grammar_blobs/css.bin grammar_blobs/go.bin grammar_blobs/html.bin grammar_blobs/java.bin grammar_blobs/javascript.bin grammar_blobs/json.bin grammar_blobs/lua.bin grammar_blobs/php.bin grammar_blobs/python.bin grammar_blobs/rust.bin grammar_blobs/sql.bin grammar_blobs/toml.bin grammar_blobs/tsx.bin grammar_blobs/typescript.bin grammar_blobs/yaml.bin
var grammarBlobFS embed.FS

func readGrammarBlob(blobName string) (grammarBlob, error) {
	data, err := grammarBlobFS.ReadFile("grammar_blobs/" + blobName)
	if err != nil {
		return grammarBlob{}, err
	}
	return grammarBlob{data: data}, nil
}
