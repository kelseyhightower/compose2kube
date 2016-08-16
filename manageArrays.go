package main

// Extend - extends the size of an array
func Extend(slice []interface{}, element interface{}) []interface{} {
	n := len(slice)
	if n == cap(slice) {
		// Slice is full; must grow.
		// We double its size and add 1, so if the size is zero we still grow.
		newSlice := make([]interface{}, len(slice), 2*len(slice)+1)
		copy(newSlice, slice)
		slice = newSlice
	}
	slice = slice[0 : n+1]
	slice[n] = element
	return slice
}

// Append - append to arrays
func Append(slice []interface{}, items ...interface{}) []interface{} {
	for _, item := range items {
		slice = Extend(slice, item)
	}
	return slice
}
