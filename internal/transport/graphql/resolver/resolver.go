package resolver

type Resolver struct {
	PostService    PostService
	UserService    UserService
	CommentService CommentService
}
