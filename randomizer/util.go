package randomizer

import (
	"fmt"
	"os"
	"reflect"
	"sort"
)

// fatal prints an error to whichever UI is used. this doesn't exit the
// program, since that would destroy the TUI.
func fatal(err error) {
	fmt.Println("fatal: ", err)
}

// a quick and dirty type of logFunc.
func printErrf(s string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, s+"\n", a...)
}

// reverseLookup looks up the key for a given map value. If multiple keys are
// associated with the same value, it will return one of those keys at random.
func reverseLookup(m, match interface{}) (interface{}, bool) {
	iter := reflect.ValueOf(m).MapRange()
	for iter.Next() {
		k, v := iter.Key(), iter.Value()
		if reflect.DeepEqual(v.Interface(), match) {
			return k.Interface(), true
		}
	}
	return nil, false
}

// guess what this does.
func reverseLookupOrPanic(m, match interface{}) interface{} {
	i, ok := reverseLookup(m, match)
	if !ok {
		panic(fmt.Sprintf("reverse lookup failed for value %v", match))
	}
	return i
}

// returns a sorted slice of string keys from a map.
func orderedKeys(m interface{}) []string {
	v := reflect.ValueOf(m)
	a := make([]string, v.Len())
	for i, key := range v.MapKeys() {
		a[i] = key.String()
	}
	sort.Strings(a)
	return a
}

// sora = Seasons OR Ages: returns the first value if the game is seasons, and
// the second if the game is ages. panics if the game is neither.
func sora(game int, sOption, aOption interface{}) interface{} {
	switch game {
	case GAME_SEASONS:
		return sOption
	case GAME_AGES:
		return aOption
	}
	panic("invalid game provided to sora()")
}

// equivalent to the ternary operation (a ? b : c) in C, etc.
func ternary(expr bool, trueOpt, falseOpt interface{}) interface{} {
	if expr {
		return trueOpt
	}
	return falseOpt
}
