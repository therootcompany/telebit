// +build tools

// tools is a faux package for tracking dependencies that don't make it into the code
package tools

import (
	_ "git.rootprojects.org/root/go-gitver"
	_ "github.com/shurcooL/vfsgen"
	_ "github.com/shurcooL/vfsgen/cmd/vfsgendev"
)
