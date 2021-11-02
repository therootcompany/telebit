//go:build !dev

//go:generate go run -mod vendor github.com/shurcooL/vfsgen/cmd/vfsgendev -source="git.rootprojects.org/root/telebit/assets/files".Assets

package files
