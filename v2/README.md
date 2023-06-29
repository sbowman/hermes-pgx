# Hermes PGX 2.1.0

Hermes PGX is an update to the https://github.com/sbowman/hermes package that wraps
https://github.com/jackc/pgx in place of the older https://github.com/lib/pq package. This package
is much lighter weight than the original Hermes, as much of the original wrapping functionality is
baked into the newer pgx package.

At its heart, Hermes PGX supplies a common interface, `hermes.Conn`, to wrap the database connection
pool and transactions so they share common functionality, and can be used interchangeably in the
context of a test or function. This makes it easier to leverage functions in different combinations
to build database APIs for Go applications.

Note that because this package is based on pgx, it only supports PostgreSQL. If you're using another
database, https://github.com/sbowman/hermes remains agnostic.

Hermes v2.1.0 upgrades support for https://github.com/jackc/pgx package to `v5`.

[![Godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/sbowman/hermes-pgx/v2)
![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)

    go get github.com/sbowman/hermes-pgx/v2

## Usage

    // Sample can take either a reference to the pgx database connection pool, or to a transaction.
    func Sample(conn hermes.Conn, name string) error {
        tx, err := conn.Begin()
        if err != nil {
            return err
        }
        
        // Will automatically rollback if an error short-circuits the return
        // before tx.Commit() is called...
        defer tx.Close() 

        res, err := conn.Exec("insert into samples (name) values ($1)", name)
        if err != nil {
            return err
        }

        check, err := res.RowsAffected()
        if check == 0 {
            return fmt.Errorf("Failed to insert row (%s)", err)
        }

        return tx.Commit()
    }

    func main() {
        // Create a connection pool with max 10 connections, min 2 idle connections...
        conn, err := hermes.Connect("postgres://postgres@127.0.0.1/my_db?sslmode=disable&connect_timeout=10")
        if err != nil {
            return err
        }

        // This works...
        if err := Sample(conn, "Bob"); err != nil {
            fmt.Println("Bob failed!", err.Error())
        }

        // So does this...
        tx, err := conn.Begin()
        if err != nil {
            panic(err)
        }

        // Will automatically rollback if call to sample fails...
        defer tx.Close() 

        if err := Sample(tx, "Frank"); err != nil {
            fmt.Println("Frank failed!", err.Error())
            return
        }

        // Don't forget to commit, or you'll automatically rollback on 
        // "defer tx.Close()" above!
        if err := tx.Commit(); err != nil {
            fmt.Println("Unable to save changes to the database:", err.Error())
        }
    }

Using a `hermes.Conn` parameter in a function also opens up *in situ* testing of database
functionality. You can create a transaction in the test case and pass it to a function that takes
a `hermes.Conn`, run any tests on the results of that function, and simply let the transaction
rollback at the end of the test to clean up.

    var DB hermes.Conn
    
    // We'll just open one database connection pool to speed up testing, so 
    // we're not constantly opening and closing connections.
    func TestMain(m *testing.M) {
	    conn, err := hermes.Connect(DBTestURI)
	    if err != nil {
	        fmt.Fprintf(os.Stderr, "Unable to open a database connection: %s\n", err)
	        os.Exit(1)
    	}
    	defer conn.Shutdown()
    	
    	DB = conn
    	
    	os.Exit(m.Run())
    }
    
    // Test getting a user account from the database.  The signature for the
    // function is:  `func GetUser(conn hermes.Conn, email string) (User, error)`
    // 
    // Passing a hermes.Conn value to the function means we can pass in either
    // a reference to the database pool and really update the data, or we can
    // pass in the same transaction reference to both the SaveUser and GetUser
    // functions.  If we use a transaction, we can let the transaction roll back 
    // after we test these functions, or at any failure point in the test case,
    // and we know the data is cleaned up. 
    func TestGetUser(t *testing.T) {
        u := User{
            Email: "jdoe@nowhere.com",
            Name: "John Doe",
        }
        
        tx, err := db.Begin()
        if err != nil {
            t.Fatal(err)
        }
        defer tx.Close()
        
        if err := tx.SaveUser(tx, u); err != nil {
            t.Fatalf("Unable to create a new user account: %s", err)
        }
        
        check, err := tx.GetUser(tx, u.Email)
        if err != nil {
            t.Fatalf("Failed to get user by email address: %s", err)
        }
        
        if check.Email != u.Email {
            t.Errorf("Expected user email to be %s; was %s", u.Email, check.Email)
        } 
        
        if check.Name != u.Name {
            t.Errorf("Expected user name to be %s; was %s", u.Name, check.Name)
        } 
        
        // Note:  do nothing...when the test case ends, the `defer tx.Close()`
        // is called, and all the data in this transaction is rolled back out.
    }

Using transactions, even if a test case fails a returns prematurely, the database transaction is
automatically closed, thanks to `defer`. The database is cleaned up without any fuss or need to
remember to delete the data you created at any point in the test.

### Shutting down the connection pool

Note that because Hermes overloads the concept of `db.Close()` and `tx.Close()`, `db.Close()`
doesn't actually do anything. In pgx, `db.Close()` would close the connection pool, which we
don't want. So instead, call `hermes.DB.Shutdown()` to clean up your connection pool when your
app shuts down.

    db, err := hermes.Connect(DBTestURI)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Unable to open a database connection: %s\n", err)
        os.Exit(1)
    }
    defer db.Shutdown()

## Advisory Locks

Hermes provides a few support functions for managing PostgreSQL advisory locks.

* `hermes.Conn.Lock` creates a session-wide, exclusive advisory lock when called on the root database pool
* `hermes.Conn.Lock` creates a transaction-wide, exclusive advisory lock when called on a transaction connection

Both functions return an `AdvisoryLock`, which should then be released to release the lock.

    db, err := hermes.Connect(DBTestURI)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Unable to open a database connection: %s\n", err)
        os.Exit(1)
    }
    defer db.Shutdown()

    // Session-wide advisory lock (lock ID = 22)
    lock, err := db.Lock(ctx, 22)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Unable to create advisory lock: %s\n", err)
        os.Exit(1)
    }
    defer lock.Release()

    tx, err := db.Begin(ctx)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Could not create a transaction: %s\n", err)
        os.Exit(1)
    }

    // This will release a transaction advisory lock
    defer tx.Close()

    // Transaction-level advisory lock
    lock, err := db.Lock(ctx, 22)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Unable to create advisory lock: %s\n", err)
        os.Exit(1)
    }
    // Technically this doesn't release the lock, but it's good practice
    defer lock.Release()

    // ...

    // This will also release the transactional advisory lock...
    tx.Commit(ctx)

Note that technically the transaction-level advisory lock doesn't require a call to `Release`; as
it will close automatically when the transaction ends. However, it's a good idea to call `Release`
regardless; that way if the `conn` is acting as a basic connection, it requires the release, and
if it's a transaction it doesn't hurt.

You may also "try" a lock, using the try functions:

    lock, err := db.TryLock(ctx, 22)
    lock, err := tx.TryLock(ctx, 22)

This will either return an advisory lock if it's available, or it will immediately return `ErrLocked`
if it's not. This can be used in situations where if one instance of an app finds the lock, it
can safely assume another instance is performing the function, such as cleaning up the database.

## Timeouts (v2.2.0)

Hermes v2.2.0 adds support for carrying connection timeout information with the `hermes.Conn`
objects. This can make it easier to create connections that don't get stuck if the database goes
away.

First, set the timeout on the `hermes.Conn` or `hermes.DB` as a default when you connect to the
database:

    db, err := hermes.Connect(DBTestURI)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Unable to open a database connection: %s\n", err)
        os.Exit(1)
    }
    defer db.Shutdown()

    db.SetTimeout(config.DBTimeout)  // if config.DBTimeout refers to a setting somewhere

Then you can leverage the `hermes.Conn.WithTimeout` method to create a timeout context and a cancel
function for you to use when making database requests:

    ctx, cancel := conn.WithTimeout(ctx) // you may also pass nil if you don't have a context
    defer cancel()

    rows, err := conn.Query(ctx, "select * from users")

Transactions also support `SetTimeout`, if you want to override the default, though it's not
typically necessary.

If you want to override the default timeout and support a longer running connection, simply pass in
your own context with a deadline and Hermes will "fake" a timeout context and simply use yours:

    // Elsewhere...
    func GetUser(ctx context.Context, conn hermes.Conn, email string) (User, error) {
        ctx, cancel := conn.WithTimeout(ctx)
        defer cancel()

        row := conn.QueryRow(ctx, "select * from users where email = $1", email)
        
        // ... load the user ...

        return user, nil
    }

    func main() {
        conn, err := hermes.Connect(DBTestURI)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Unable to open a database connection: %s\n", err)
            os.Exit(1)
        }
        defer conn.Shutdown()

        conn.SetTimeout(time.Second)  

        // For some reason it takes a long time to get a user...
        ctx, cancel := context.WithTimeout(ctx, time.Minute)
        defer cancel()

        // If the default for db is 1 second, but the context is set to timeout in a minute, this
        // call may take as long as a minute:
        user, err := GetUser(ctx, conn, "jdoe@nowhere.com")

### ContextualTx prototype

There's also a "contextual" transaction that tries to manage the timeout for you. It's experimental,
but may be worth a look. Simply call `conn.BeginWithTimeout` rather than `conn.Begin` to create
a transaction. You can then skip passing in a context to every request and use the context
maintained internally in the transaction:

    tx, err := conn.BeginWithTimeout(ctx) // if ctx already has a deadline, that deadline is used
    if err != nil {
        return err
    }
    defer tx.Close() // this will cancel the timeout context

    var userID int
    row := tx.QueryRow("select id from users where email = $1", email)
    if err := row.Scan(&userID); err != nil {
        return err
    }

    tx.Exec("insert into admin_users values ($1)", userID
    return tx.Commit()

If `ctx` above is nil or doesn't have a deadline, `BeginWithTimeout` will use the default timeout
and create a context it carries around with the transaction. If `ctx` does have a deadline, it'll
use that existing context as the underlying context. Every database request will have that context
attached to it automatically, and when you call `tx.Close()` the `context.CancelFunc` is called
as prescribed by the `context` package.

## Deprecated

The pgx package changes how custom types are handled.

## References

https://github.com/jackc/pgx
