package context

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// TxContext is a regular context, with additional references to a firestore.Transaction
// which can be used for reading and writing data. If all firestore getters & setters use
// this structure, we can be sure that our reads and writes are always atomic.
type TxContext struct {
	context.Context
	t time.Time
	// operations should return error if either of these are nil - otherwise, just wraps a few of their methods
	tx  *firestore.Transaction
	cli *firestore.Client
}

// New creates a context which will execute all of its operations using a firestore transaction.
func New(ctx context.Context, t time.Time, tx *firestore.Transaction, cli *firestore.Client,
) TxContext {
	return TxContext{
		Context: ctx,
		t:       t,
		tx:      tx,
		cli:     cli,
	}
}

// Now creates a context which will execute all of its operations using a firestore transaction.
// With the time as time.Now.
func Now(ctx context.Context, tx *firestore.Transaction, cli *firestore.Client,
) TxContext {
	return New(ctx, time.Now(), tx, cli)
}

// Time returns the contexts's stored time.
func (ctx TxContext) Time() time.Time {
	return ctx.t
}

// UnixTime returns the contexts's stored unix time as an int.
func (ctx TxContext) UnixTime() int {
	return int(ctx.t.Unix())
}

// Get runs a Get operation in the context's transaction.
func (ctx TxContext) Get(dr *firestore.DocumentRef) (*firestore.DocumentSnapshot, error) {
	if ctx.tx == nil {
		return nil, errors.New("TxContext not initialized")
	}
	if err := ctx.Err(); err != nil {
		return nil, err // respect context cancellation or deadline
	}
	return ctx.tx.Get(dr)
}

// Delete runs a Delete operation in the context's transaction.
func (ctx TxContext) Delete(dr *firestore.DocumentRef) error {
	if ctx.tx == nil {
		return errors.New("TxContext not initialized")
	}
	if err := ctx.Err(); err != nil {
		return err // respect context cancellation or deadline
	}
	return ctx.tx.Delete(dr) // note: no-op with no error if document not found
}

// DeleteCollection deletes all documents found in a collection.
// Note that this contains both a read and a write operation, which restricts
// when it can occur during firestore transactions.
func (ctx TxContext) DeleteCollection(cr *firestore.CollectionRef) error {
	if ctx.tx == nil {
		return errors.New("TxContext not initialized")
	}
	if err := ctx.Err(); err != nil {
		return err // respect context cancellation or deadline
	}
	collIter := ctx.tx.Documents(cr)
	for {
		doc, err := collIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		if err := ctx.Delete(doc.Ref); err != nil {
			return err
		}
	}
	return nil
}

// Set runs a Set operation in the context's transaction.
func (ctx TxContext) Set(dr *firestore.DocumentRef, data interface{}, opts ...firestore.SetOption) error {
	if ctx.tx == nil {
		return errors.New("TxContext not initialized")
	}
	if err := ctx.Err(); err != nil {
		return err // respect context cancellation or deadline
	}
	return ctx.tx.Set(dr, data, opts...)
}

// Documents creates a document iterator from the context's transaction. Note that creating
// this iterator is a read operation, so it cannot occur after any writes.
func (ctx TxContext) Documents(q firestore.Queryer) (*firestore.DocumentIterator, error) {
	if ctx.tx == nil {
		return nil, errors.New("TxContext not initialized")
	}
	if err := ctx.Err(); err != nil {
		return nil, err // respect context cancellation or deadline
	}
	return ctx.tx.Documents(q), nil
}

// Collection returns a collection.
func (ctx TxContext) Collection(path string) *firestore.CollectionRef {
	if ctx.cli == nil {
		return nil
	}
	return ctx.cli.Collection(path)
}

// CollectionGroup returns a collection group.
func (ctx TxContext) CollectionGroup(path string) *firestore.CollectionGroupRef {
	if ctx.cli == nil {
		return nil
	}
	return ctx.cli.CollectionGroup(path)
}
