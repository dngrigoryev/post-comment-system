package loader

import (
	"context"

	"client-services/internal/domain"

	"github.com/99designs/gqlgen/graphql"
)

type selectionInfo struct {
	pageKey
	author bool
}

func PrefetchPostPage(ctx context.Context, page *domain.PostPage) {
	if !graphql.HasOperationContext(ctx) {
		return
	}
	op := graphql.GetOperationContext(ctx)
	fields := nodeFields(op, graphql.CollectFieldsCtx(ctx, []string{"PostPage"}), "Post")
	prefetchPosts(ctx, page.Nodes, fields)
}

func PrefetchPost(ctx context.Context, post *domain.Post) {
	if !graphql.HasOperationContext(ctx) {
		return
	}
	prefetchPosts(ctx, []domain.Post{*post}, graphql.CollectFieldsCtx(ctx, []string{"Post"}))
}

func prefetchPosts(ctx context.Context, posts []domain.Post, fields []graphql.CollectedField) {
	roots := make([]domain.CommentTreeRoot, len(posts))
	var authors []int64
	for i, post := range posts {
		roots[i] = domain.CommentTreeRoot{PostID: post.ID}
		if hasAuthor(fields) {
			authors = append(authors, post.AuthorID)
		}
	}
	prefetch(ctx, roots, fields, "comments", authors)
}

func PrefetchComment(ctx context.Context, comment *domain.Comment) {
	if !graphql.HasOperationContext(ctx) {
		return
	}
	fields := graphql.CollectFieldsCtx(ctx, []string{"Comment"})
	var authors []int64
	if hasAuthor(fields) {
		authors = append(authors, comment.AuthorID)
	}
	prefetch(ctx, []domain.CommentTreeRoot{{PostID: comment.PostID, ParentCommentID: &comment.ID}}, fields, "replies", authors)
}

func prefetch(ctx context.Context, roots []domain.CommentTreeRoot, fields []graphql.CollectedField, fieldName string, authorIDs []int64) {
	value, err := fromContext(ctx)
	if err != nil || len(roots) == 0 {
		return
	}
	op := graphql.GetOperationContext(ctx)
	info := make(map[int]selectionInfo)
	plan := commentSelections(op, fields, fieldName, info)
	if len(plan) > 0 {
		pages, err := value.commentService.LoadCommentTree(ctx, domain.CommentTreeRequest{Roots: roots, Selections: plan})

		if err == nil {
			for _, loaded := range pages {
				selection := info[loaded.SelectionID]
				if loaded.ParentCommentID == nil {
					value.postComments.Prime(postCommentsKey{PostID: loaded.PostID, pageKey: selection.pageKey}, loaded.Page)
				} else {
					value.replies.Prime(repliesKey{PostID: loaded.PostID, ParentCommentID: *loaded.ParentCommentID, pageKey: selection.pageKey}, loaded.Page)
				}
				if selection.author {
					for _, comment := range loaded.Page.Nodes {
						authorIDs = append(authorIDs, comment.AuthorID)
					}
				}
			}
		}
	}
	seen := make(map[int64]bool, len(authorIDs))
	ids := make([]int64, 0, len(authorIDs))
	for _, id := range authorIDs {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}
	users, errs := value.userService.GetUsersBatch(ctx, ids)
	for i, id := range ids {
		if errs[i] == nil {
			value.users.Prime(id, users[i])
		}
	}
}

func commentSelections(op *graphql.OperationContext, fields []graphql.CollectedField, name string, info map[int]selectionInfo) []domain.CommentSelection {
	var keys []pageKey
	nodes := make(map[pageKey][]graphql.CollectedField)
	for _, field := range fields {
		if field.Name != name {
			continue
		}
		args := field.ArgumentMap(op.Variables)
		first, err := graphql.UnmarshalInt(args["first"])
		if err != nil {
			continue
		}
		var after *string
		if args["after"] != nil {
			text, err := graphql.UnmarshalString(args["after"])
			if err != nil {
				continue
			}
			after = &text
		}
		key := newPageKey(first, after)
		if _, ok := nodes[key]; !ok {
			keys = append(keys, key)
			nodes[key] = nil
		}
		pageFields := graphql.CollectFields(op, field.Selections, []string{"CommentPage"})
		nodes[key] = append(nodes[key], nodeFields(op, pageFields, "Comment")...)
	}
	selections := make([]domain.CommentSelection, 0, len(keys))
	for _, key := range keys {
		fields := nodes[key]
		id := len(info) + 1
		info[id] = selectionInfo{pageKey: key, author: hasAuthor(fields)}
		selections = append(selections, domain.CommentSelection{
			ID: id, Page: key.request(), Replies: commentSelections(op, fields, "replies", info),
		})
	}
	return selections
}

func nodeFields(op *graphql.OperationContext, fields []graphql.CollectedField, nodeType string) []graphql.CollectedField {
	var nodes []graphql.CollectedField
	for _, field := range fields {
		if field.Name == "nodes" {

			nodes = append(nodes, graphql.CollectFields(op, field.Selections, []string{nodeType})...)
		}
	}
	return nodes
}

func hasAuthor(fields []graphql.CollectedField) bool {
	for _, field := range fields {
		if field.Name == "author" {
			return true
		}
	}
	return false
}
