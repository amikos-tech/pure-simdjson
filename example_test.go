package purejson

import (
	"errors"
	"fmt"
)

func ExampleParser_Parse() {
	parser, err := NewParser()
	if err != nil {
		panic(err)
	}
	defer func() { _ = parser.Close() }()

	doc, err := parser.Parse([]byte(`42`))
	if err != nil {
		panic(err)
	}
	defer func() { _ = doc.Close() }()

	value, err := doc.Root().GetInt64()
	if err != nil {
		panic(err)
	}

	fmt.Println(value)
	// Output: 42
}

func ExampleDoc_Root() {
	parser, err := NewParser()
	if err != nil {
		panic(err)
	}
	defer func() { _ = parser.Close() }()

	doc, err := parser.Parse([]byte(`{"name":"alice"}`))
	if err != nil {
		panic(err)
	}
	defer func() { _ = doc.Close() }()

	object, err := doc.Root().AsObject()
	if err != nil {
		panic(err)
	}

	name, err := object.GetStringField("name")
	if err != nil {
		panic(err)
	}

	fmt.Println(name)
	// Output: alice
}

func ExampleElement_scalarAccess() {
	parser, err := NewParser()
	if err != nil {
		panic(err)
	}
	defer func() { _ = parser.Close() }()

	doc, err := parser.Parse([]byte(`{"id":7,"name":"alice","active":true}`))
	if err != nil {
		panic(err)
	}
	defer func() { _ = doc.Close() }()

	object, err := doc.Root().AsObject()
	if err != nil {
		panic(err)
	}

	idField, err := object.GetField("id")
	if err != nil {
		panic(err)
	}
	id, err := idField.GetInt64()
	if err != nil {
		panic(err)
	}

	nameField, err := object.GetField("name")
	if err != nil {
		panic(err)
	}
	name, err := nameField.GetString()
	if err != nil {
		panic(err)
	}

	activeField, err := object.GetField("active")
	if err != nil {
		panic(err)
	}
	active, err := activeField.GetBool()
	if err != nil {
		panic(err)
	}

	fmt.Println(id, name, active)
	// Output: 7 alice true
}

func ExampleElementType() {
	parser, err := NewParser()
	if err != nil {
		panic(err)
	}
	defer func() { _ = parser.Close() }()

	doc, err := parser.Parse([]byte(`18446744073709551615`))
	if err != nil {
		panic(err)
	}
	defer func() { _ = doc.Close() }()

	fmt.Println(doc.Root().Type() == TypeUint64)
	// Output: true
}

func ExampleArray_Iter() {
	parser, err := NewParser()
	if err != nil {
		panic(err)
	}
	defer func() { _ = parser.Close() }()

	doc, err := parser.Parse([]byte(`[1,2,3]`))
	if err != nil {
		panic(err)
	}
	defer func() { _ = doc.Close() }()

	array, err := doc.Root().AsArray()
	if err != nil {
		panic(err)
	}

	sum := int64(0)
	iter := array.Iter()
	for iter.Next() {
		value, err := iter.Value().GetInt64()
		if err != nil {
			panic(err)
		}
		sum += value
	}
	if err := iter.Err(); err != nil {
		panic(err)
	}

	fmt.Println(sum)
	// Output: 6
}

func ExampleArrayIter_Next() {
	parser, err := NewParser()
	if err != nil {
		panic(err)
	}
	defer func() { _ = parser.Close() }()

	doc, err := parser.Parse([]byte(`["first","second"]`))
	if err != nil {
		panic(err)
	}
	defer func() { _ = doc.Close() }()

	array, err := doc.Root().AsArray()
	if err != nil {
		panic(err)
	}

	iter := array.Iter()
	for iter.Next() {
		value, err := iter.Value().GetString()
		if err != nil {
			panic(err)
		}
		fmt.Println(value)
	}
	fmt.Println(iter.Err() == nil)
	// Output:
	// first
	// second
	// true
}

func ExampleObject_Iter() {
	parser, err := NewParser()
	if err != nil {
		panic(err)
	}
	defer func() { _ = parser.Close() }()

	doc, err := parser.Parse([]byte(`{"a":1,"b":2}`))
	if err != nil {
		panic(err)
	}
	defer func() { _ = doc.Close() }()

	object, err := doc.Root().AsObject()
	if err != nil {
		panic(err)
	}

	iter := object.Iter()
	for iter.Next() {
		value, err := iter.Value().GetInt64()
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s=%d\n", iter.Key(), value)
	}
	if err := iter.Err(); err != nil {
		panic(err)
	}
	// Output:
	// a=1
	// b=2
}

func ExampleObject_GetField() {
	parser, err := NewParser()
	if err != nil {
		panic(err)
	}
	defer func() { _ = parser.Close() }()

	doc, err := parser.Parse([]byte(`{"active":true}`))
	if err != nil {
		panic(err)
	}
	defer func() { _ = doc.Close() }()

	object, err := doc.Root().AsObject()
	if err != nil {
		panic(err)
	}

	field, err := object.GetField("active")
	if err != nil {
		panic(err)
	}

	active, err := field.GetBool()
	if err != nil {
		panic(err)
	}

	fmt.Println(active)
	// Output: true
}

func ExampleObject_GetStringField() {
	parser, err := NewParser()
	if err != nil {
		panic(err)
	}
	defer func() { _ = parser.Close() }()

	doc, err := parser.Parse([]byte(`{"name":"alice"}`))
	if err != nil {
		panic(err)
	}
	defer func() { _ = doc.Close() }()

	object, err := doc.Root().AsObject()
	if err != nil {
		panic(err)
	}

	name, err := object.GetStringField("name")
	if err != nil {
		panic(err)
	}

	fmt.Println(name)
	// Output: alice
}

func ExampleObjectIter_Next() {
	parser, err := NewParser()
	if err != nil {
		panic(err)
	}
	defer func() { _ = parser.Close() }()

	doc, err := parser.Parse([]byte(`{"name":"alice"}`))
	if err != nil {
		panic(err)
	}
	defer func() { _ = doc.Close() }()

	object, err := doc.Root().AsObject()
	if err != nil {
		panic(err)
	}

	iter := object.Iter()
	fmt.Println(iter.Next())
	fmt.Println(iter.Key())
	value, err := iter.Value().GetString()
	if err != nil {
		panic(err)
	}
	fmt.Println(value)
	fmt.Println(iter.Next())
	fmt.Println(iter.Err() == nil)
	// Output:
	// true
	// name
	// alice
	// false
	// true
}

func ExampleParserPool_Get() {
	pool := NewParserPool()

	parser, err := pool.Get()
	if err != nil {
		panic(err)
	}

	doc, err := parser.Parse([]byte(`{"status":"ok"}`))
	if err != nil {
		panic(err)
	}

	object, err := doc.Root().AsObject()
	if err != nil {
		panic(err)
	}

	status, err := object.GetStringField("status")
	if err != nil {
		panic(err)
	}

	if err := doc.Close(); err != nil {
		panic(err)
	}
	if err := pool.Put(parser); err != nil {
		panic(err)
	}

	fmt.Println(status)
	// Output: ok
}

func ExampleError() {
	parser, err := NewParser()
	if err != nil {
		panic(err)
	}
	defer func() { _ = parser.Close() }()

	_, err = parser.Parse([]byte(`{"name":`))
	if err == nil {
		panic("expected parse error")
	}

	fmt.Println(errors.Is(err, ErrInvalidJSON))

	var nativeErr *Error
	fmt.Println(errors.As(err, &nativeErr))
	// Output:
	// true
	// true
}
