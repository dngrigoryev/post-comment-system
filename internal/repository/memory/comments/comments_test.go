package comments

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"client-services/internal/domain"
	"client-services/internal/repository/cursor"
	"client-services/internal/repository/memory/internal/store"
	"client-services/internal/repository/memory/posts"
)

func fixture(t *testing.T) (*CommentRepository, *posts.PostRepository, int64) {
	t.Helper()
	s := store.New(domain.User{ID: 1}, domain.User{ID: 2})
	p := posts.New(s)
	post, err := p.CreatePost(t.Context(), domain.Post{AuthorID: 1, Title: "post", CommentsPermitted: true})
	if err != nil {
		t.Fatal(err)
	}
	return New(s), p, post.ID
}
func add(t *testing.T, r *CommentRepository, post int64, parent *int64) *domain.Comment {
	t.Helper()
	c, err := r.AddComment(t.Context(), domain.Comment{PostID: post, ParentCommentID: parent, AuthorID: 1, Content: "text"})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRootsRepliesPaginationAndCopyIsolation(t *testing.T) {
	r, _, post := fixture(t)
	root := add(t, r, post, nil)
	reply := add(t, r, post, &root.ID)
	second := add(t, r, post, nil)
	add(t, r, post, &root.ID)
	grandchild := add(t, r, post, &reply.ID)
	page, err := r.ListPostComments(t.Context(), post, domain.PageRequest{First: 1})
	if err != nil || len(page.Nodes) != 1 || page.Nodes[0].ID != root.ID || !page.PageInfo.HasNextPage {
		t.Fatal(page, err)
	}
	page, err = r.ListPostComments(t.Context(), post, domain.PageRequest{First: 1, After: page.PageInfo.EndCursor})
	if err != nil || page.Nodes[0].ID != second.ID || page.PageInfo.HasNextPage {
		t.Fatal(page, err)
	}
	page, err = r.ListReplies(t.Context(), root.ID, domain.PageRequest{First: 1})
	if err != nil || page.Nodes[0].ID != reply.ID || !page.PageInfo.HasNextPage {
		t.Fatal(page, err)
	}
	page, err = r.ListReplies(t.Context(), root.ID, domain.PageRequest{First: 1, After: page.PageInfo.EndCursor})
	if err != nil || len(page.Nodes) != 1 || page.PageInfo.HasNextPage {
		t.Fatal(page, err)
	}
	page, err = r.ListReplies(t.Context(), reply.ID, domain.PageRequest{First: 1})
	if err != nil || page.Nodes[0].ID != grandchild.ID {
		t.Fatal(page, err)
	}
	parentInput := root.ID
	c := add(t, r, post, &parentInput)
	parentInput = 999
	*c.ParentCommentID = 998
	stored, err := r.GetComment(t.Context(), c.ID)
	if err != nil || *stored.ParentCommentID != root.ID {
		t.Fatal(stored, err)
	}
	*stored.ParentCommentID = 997
	page, _ = r.ListReplies(t.Context(), root.ID, domain.PageRequest{First: 100})
	for i := range page.Nodes {
		*page.Nodes[i].ParentCommentID = 996
	}
	stored, _ = r.GetComment(t.Context(), c.ID)
	if *stored.ParentCommentID != root.ID {
		t.Fatal("page/get result mutated storage")
	}
	empty, err := r.ListReplies(t.Context(), second.ID, domain.PageRequest{First: 1})
	if err != nil || empty.Nodes == nil || len(empty.Nodes) != 0 || empty.PageInfo.EndCursor != nil {
		t.Fatal(empty, err)
	}
}

func TestReferenceErrorsAndDisabledComments(t *testing.T) {
	r, p, post := fixture(t)
	other, err := p.CreatePost(t.Context(), domain.Post{AuthorID: 1, CommentsPermitted: true})
	if err != nil {
		t.Fatal(err)
	}
	foreign := add(t, r, other.ID, nil)
	for _, tc := range []struct {
		comment domain.Comment
		want    error
	}{
		{domain.Comment{PostID: post, AuthorID: 99}, domain.ErrInvalidArgument},
		{domain.Comment{PostID: 999, AuthorID: 1}, domain.ErrNotFound},
		{domain.Comment{PostID: post, AuthorID: 1, ParentCommentID: &foreign.ID}, domain.ErrInvalidArgument},
	} {
		_, err := r.AddComment(t.Context(), tc.comment)
		if !errors.Is(err, tc.want) {
			t.Fatal(err)
		}
	}
	_, err = r.GetComment(t.Context(), 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	_, err = r.ListReplies(t.Context(), 999, domain.PageRequest{First: 1})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	_, err = p.DisableComments(t.Context(), 1, post)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.AddComment(t.Context(), domain.Comment{PostID: post, AuthorID: 1})
	if !errors.Is(err, domain.ErrCommentsDisabled) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = r.GetComment(ctx, foreign.ID)
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestBatchCursorScopesAndIndependentErrors(t *testing.T) {
	r, p, post := fixture(t)
	root := add(t, r, post, nil)
	child := add(t, r, post, &root.ID)
	add(t, r, post, &root.ID)
	other, _ := p.CreatePost(t.Context(), domain.Post{AuthorID: 1, CommentsPermitted: true})
	otherRoot := add(t, r, other.ID, nil)
	after, _ := cursor.EncodeComment(child.ID, post, &root.ID)
	wrong, _ := cursor.EncodeComment(root.ID, post, nil)
	pages, errs := r.ListRepliesBatch(t.Context(), []domain.RepliesRequest{
		{PostID: post, ParentCommentID: root.ID, Page: domain.PageRequest{First: 1}},
		{PostID: post, ParentCommentID: root.ID, Page: domain.PageRequest{First: 1, After: &after}},
		{PostID: other.ID, ParentCommentID: root.ID, Page: domain.PageRequest{First: 1}},
		{PostID: post, ParentCommentID: root.ID, Page: domain.PageRequest{First: 1, After: &wrong}},
	})
	if errs[0] != nil || errs[1] != nil || !errors.Is(errs[2], domain.ErrNotFound) || !errors.Is(errs[3], domain.ErrInvalidArgument) || pages[0].Nodes[0].ID != child.ID || pages[1].PageInfo.HasNextPage {
		t.Fatal(pages, errs)
	}
	*pages[0].Nodes[0].ParentCommentID = 999
	stored, _ := r.GetComment(t.Context(), child.ID)
	if *stored.ParentCommentID != root.ID {
		t.Fatal("batch aliased storage")
	}
	pages, errs = r.ListPostCommentsBatch(t.Context(), []domain.PostCommentsRequest{{PostID: other.ID, Page: domain.PageRequest{First: 1}}, {PostID: 999, Page: domain.PageRequest{First: 1}}, {PostID: post, Page: domain.PageRequest{First: 0}}})
	if errs[0] != nil || pages[0].Nodes[0].ID != otherRoot.ID || !errors.Is(errs[1], domain.ErrNotFound) || !errors.Is(errs[2], domain.ErrInvalidArgument) {
		t.Fatal(pages, errs)
	}
}

func TestTreeFollowsOnlyReturnedNodes(t *testing.T) {
	r, _, post := fixture(t)
	root := add(t, r, post, nil)
	hidden := add(t, r, post, nil)
	child := add(t, r, post, &root.ID)
	add(t, r, post, &hidden.ID)
	add(t, r, post, &root.ID)
	request := domain.CommentTreeRequest{Roots: []domain.CommentTreeRoot{{PostID: post}}, Selections: []domain.CommentSelection{{ID: 1, Page: domain.PageRequest{First: 1}, Replies: []domain.CommentSelection{{ID: 2, Page: domain.PageRequest{First: 1}, Replies: []domain.CommentSelection{{ID: 3, Page: domain.PageRequest{First: 1}}}}}}}}
	pages, err := r.LoadCommentTree(t.Context(), request)
	if err != nil || len(pages) != 3 {
		t.Fatal(pages, err)
	}
	for _, p := range pages {
		switch p.SelectionID {
		case 1:
			if !p.Page.PageInfo.HasNextPage || p.Page.Nodes[0].ID != root.ID {
				t.Fatal(p)
			}
		case 2:
			if *p.ParentCommentID != root.ID || p.Page.Nodes[0].ID != child.ID || !p.Page.PageInfo.HasNextPage {
				t.Fatal(p)
			}
		case 3:
			if *p.ParentCommentID != child.ID || len(p.Page.Nodes) != 0 {
				t.Fatal(p)
			}
		}
	}

	parent := child.ID
	for i := 0; i < 1000; i++ {
		c := add(t, r, post, &parent)
		parent = c.ID
	}
	var plan []domain.CommentSelection
	for i := 1000; i > 0; i-- {
		plan = []domain.CommentSelection{{ID: i, Page: domain.PageRequest{First: 1}, Replies: plan}}
	}
	pages, err = r.LoadCommentTree(t.Context(), domain.CommentTreeRequest{Roots: []domain.CommentTreeRoot{{PostID: post, ParentCommentID: &child.ID}}, Selections: plan})
	if err != nil || len(pages) != 1000 {
		t.Fatalf("pages=%d err=%v", len(pages), err)
	}
}

func TestConcurrentIndexUpdatesAndReads(t *testing.T) {
	r, p, post := fixture(t)
	parent := add(t, r, post, nil)
	var wg sync.WaitGroup
	failures := make(chan error, 1024)
	for i := 0; i < 80; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var parentID *int64
			if i%2 == 0 {
				parentID = &parent.ID
			}
			_, err := r.AddComment(t.Context(), domain.Comment{PostID: post, ParentCommentID: parentID, AuthorID: 1})
			if err != nil {
				failures <- err
			}
		}(i)
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				pages, errs := r.ListRepliesBatch(t.Context(), []domain.RepliesRequest{{PostID: post, ParentCommentID: parent.ID, Page: domain.PageRequest{First: 5}}})
				if errs[0] != nil {
					failures <- errs[0]
				} else {
					for k := 1; k < len(pages[0].Nodes); k++ {
						if pages[0].Nodes[k-1].ID >= pages[0].Nodes[k].ID {
							failures <- fmt.Errorf("unsorted reply index")
						}
					}
				}
				_, err := r.LoadCommentTree(t.Context(), domain.CommentTreeRequest{Roots: []domain.CommentTreeRoot{{PostID: post}}, Selections: []domain.CommentSelection{{ID: 1, Page: domain.PageRequest{First: 5}, Replies: []domain.CommentSelection{{ID: 2, Page: domain.PageRequest{First: 5}}}}}})
				if err != nil {
					failures <- err
				}
			}
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	page, err := r.ListReplies(t.Context(), parent.ID, domain.PageRequest{First: 100})
	if err != nil || len(page.Nodes) != 40 {
		t.Fatal(page, err)
	}
	page, err = r.ListPostComments(t.Context(), post, domain.PageRequest{First: 100})
	if err != nil || len(page.Nodes) != 41 {
		t.Fatal(page, err)
	}

	_, err = p.DisableComments(t.Context(), 1, post)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.AddComment(t.Context(), domain.Comment{PostID: post, AuthorID: 1})
	if !errors.Is(err, domain.ErrCommentsDisabled) {
		t.Fatal(err)
	}
}

func TestSeparateMutexes(t *testing.T) {
	r, p, post := fixture(t)
	c := add(t, r, post, nil)
	r.state.CommentsMu.Lock()
	done := make(chan error, 1)
	go func() { _, err := p.CreatePost(t.Context(), domain.Post{AuthorID: 1}); done <- err }()
	select {
	case err := <-done:
		r.state.CommentsMu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		r.state.CommentsMu.Unlock()
		t.Fatal("comment lock blocked post write")
	}
	r.state.PostsMu.Lock()
	go func() { _, err := r.GetComment(t.Context(), c.ID); done <- err }()
	select {
	case err := <-done:
		r.state.PostsMu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		r.state.PostsMu.Unlock()
		t.Fatal("post lock blocked comment read")
	}
}
