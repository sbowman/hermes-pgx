# Hermes PGX 1.0.0

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

[![Godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/sbowman/hermes-pgx)
![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)

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
    	defer conn.Close()
    	
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

## References

https://github.com/jackc/pgx
