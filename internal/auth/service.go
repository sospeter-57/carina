/* this header file contains utilities for hashing
 * plain text passwords before storing in the database
 * as well as comparing sign in password hash with 
 * theh one stored in the database
 * both instances are authentication, i choose bcrypt pakge for this
 */

package auth

import (
	"golang.org/x/crypto/bcrypt"
	"errors"
)

// return the hash as a result of our plaintext HashPassword
// also takes in the cost, to vary work factor (10/12/14)
// a cost/work factor of 12 is standard
// if a cost of less than minimum(10) is provided, cost becomes the default
func HashPassword(plainTextPassword []byte, cost int) ([]byte, error) {
	// first we check invalid cost parsed in
	if cost < 12 || cost > 14 {
		costErr := errors.New("cost error can't be less than 10 or more than 14")
		return []byte{}, costErr
	}
	
	passwordHash, err := bcrypt.GenerateFromPassword(plainTextPassword, cost)
	if err != nil {
		return []byte{}, err
	}

	return passwordHash, nil
}

// in another case we might want to compare input password with 
// the hash in our database
// we'd query our database for the hashed password, the pass
// to this function in addition to the input one to compare
// this is just a wrapper function to isolate concerns

func VerifyPassword(hashed, plaintext []byte) error {
	err := bcrypt.CompareHashAndPassword(hashed, plaintext)
	return err
}
