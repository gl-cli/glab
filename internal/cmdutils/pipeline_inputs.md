Specify one or more pipeline inputs using the `-i` or `--input` flag for each
input. Each input flag uses the format `key:value`.

The values are typed and will default to `string` unless a type is explicitly
specified. To specify a type, use the `type(value)` syntax. For example,
`key:string(value)` will pass the string `value` as the input.

Valid types are:

- `string`: A string value. This is the default type. For example, `key:string(value)`.
- `int`: An integer value. For example, `key:int(42)`.
- `float`: A floating-point value. For example, `key:float(3.14)`.
- `bool`: A boolean value. For example, `key:bool(true)`.
- `array`: An array of strings. For example, `key:array(foo,bar)`.

An array of strings can be specified with a trailing comma. For example,
`key:array(foo,bar,)` will pass the array `[foo, bar]`. `array()` specifies an
empty array. To pass an array with the empty string, use `array(,)`.

Value arguments containing parentheses should be escaped from the shell with
quotes. For example, `--input key:array(foo,bar)` should be written as
`--input 'key:array(foo,bar)'`.
