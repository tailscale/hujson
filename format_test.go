// Copyright (c) 2021 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hujson

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFormatErrors(t *testing.T) {
	tests := []struct {
		name   string
		format func([]byte) ([]byte, error)
	}{
		{"Standardize", Standardize},
		{"Minimize", Minimize},
		{"Format", Format},
	}

	const want = "[null,false,true,invalid]"
	for _, tt := range tests {
		got, err := tt.format([]byte(want))
		if err == nil {
			t.Errorf("%s error = nil, want non-nil", tt.name)
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", tt.name, got, want)
		}
	}
}

var testdataFormat = []struct {
	in   string
	want string
}{{
	in:   `null`,
	want: `null`,
}, {
	in:   " \r\n\t//comment\n\n\n/**/null \r\t//comment\n\n\n/**/\r\n\t",
	want: "//comment\n\n/**/ null //comment\n\n/**/",
}, {
	in:   "//comment\r\n//comment\n\r/**\r/*/null",
	want: "//comment\n//comment\n/** /*/ null",
}, {
	in:   `"\u000F\u000a\/\ud83d\ude02"`,
	want: `"\u000f\n/😂"`,
}, {
	in:   "{\n\r\t \n\r\t }",
	want: "{}",
}, {
	in:   "{/**/}",
	want: "{ /**/ }",
}, {
	in:   "{//\r\t\n}",
	want: "{ //\n}",
}, {
	in:   "[\n\r\t \n\r\t ]",
	want: "[]",
}, {
	in:   "[/**/]",
	want: "[ /**/ ]",
}, {
	in:   "[//\r\t\n]",
	want: "[ //\n]",
}, {
	in:   `{"name" 	 	:"value" 	 	,"name":"value"}`,
	want: `{"name": "value", "name": "value"}`,
}, {
	in:   `{"name"/**/:"value"/**/,"name":"value"}`,
	want: `{"name" /**/ : "value" /**/ , "name": "value"}`,
}, {
	in:   `[null 	 	,null]`,
	want: `[null, null]`,
}, {
	in:   `[null/**/,null]`,
	want: `[null /**/ , null]`,
}, {
	in:   "[0//\n,]",
	want: "\n[\n\t0 //\n\t\t,\n]",
}, {
	in:   "[/*\n*/\n]",
	want: "[ /*\n\t */\n]",
}, {
	in:   "[/*\n\n*/\n]",
	want: "[ /*\n\n\t*/\n]",
}, {
	in:   "[ /*\n\t\n\t*/\n]",
	want: "[ /*\n\n\t*/\n]",
}, {
	in: `[
			/*

		line1
  line2

			*/

		]`,
	want: `
[
	/*

			line1
	  line2

	*/
]`,
}, {
	in: `[
		/*

  	line1
  line2

		*/

	]`,
	want: `
[
	/*

		line1
	line2

	*/
]`,
}, {
	in: `[
/*
* line1
* line2
*/
		]`,
	want: `
[
	/*
	 * line1
	 * line2
	 */
]`,
}, {
	in: `[
/*
	* line1
* line2
*/
	]`,
	want: `
[
	/*
		* line1
	* line2
	*/
]`,
}, {
	in: "//😊 \r\t☹\n/*😊 \r\t☹\n*/null//😊 \r\t\n/*\r\t\n*/",
	want: `
//😊  	☹
/*😊  	☹
 */ null //😊
/*
 */`,
}, {
	in: `
		
		   // LineComment   
		/* 
  BlockComment     
			 */

	{
		
		// LineComment   
	 /* 
BlockComment     
		  */

"name"

		
// LineComment   
/* 
BlockComment     
	 */

:
		
    // LineComment   
/* 
 BlockComment     
	 */

			   "value"

		
         			   // LineComment   
		   	   /* 
			                         BlockComment     
					*/

			   ,

		
	    		   // LineComment      
   			   /* 
			   BlockComment     
					*/
			   

	}

			
// LineComment   
/* 
BlockComment     
	 */



	 `,
	want: `
// LineComment
/*
BlockComment
*/

{
	// LineComment
	/*
	BlockComment
	*/

	"name"
		// LineComment
		/*
		BlockComment
		*/
		:
		// LineComment
		/*
		BlockComment
		*/
		"value"
		// LineComment
		/*
		BlockComment
		*/
		,

	// LineComment
	/*
	BlockComment
	*/
}

// LineComment
/*
BlockComment
*/`,
}, {
	in: `
		//line1
		{//line2
		"name"//line3
		://line4
		"value"//line5
		}//line6
		`,
	want: `
//line1
{ //line2
	"name" //line3
		: //line4
		"value", //line5
} //line6`,
}, {
	in:   `/**//**/{/**//**/"name"/**//**/:/**//**/null/**//**/,}/**//**/`,
	want: `/**/ /**/ { /**/ /**/ "name" /**/ /**/ : /**/ /**/ null /**/ /**/ } /**/ /**/`,
}, {
	in:   `/**//**/{/**//**/"name"/**//**/:/**//**/null/**//**/,/**//**/}/**//**/`,
	want: `/**/ /**/ { /**/ /**/ "name" /**/ /**/ : /**/ /**/ null /**/ /**/ , /**/ /**/ } /**/ /**/`,
}, {
	in:   `/**//**/[/**//**/null/**//**/,]/**//**/`,
	want: `/**/ /**/ [ /**/ /**/ null /**/ /**/ ] /**/ /**/`,
}, {
	in:   `/**//**/[/**//**/null/**//**/,/**//**/]/**//**/`,
	want: `/**/ /**/ [ /**/ /**/ null /**/ /**/ , /**/ /**/ ] /**/ /**/`,
}, {
	in: `{
				"name": "value",
				"name______": "value",
				"name_": "value",
				"name___": "value"
			}`,
	want: `
{
	"name":       "value",
	"name______": "value",
	"name_":      "value",
	"name___":    "value"
}`,
}, {
	in: `{
		"name": "value",
		"name______": "value",
		// comment
		"name_": "value",
		"name___": "value"
		}`,
	want: `
{
	"name":       "value",
	"name______": "value",
	// comment
	"name_":   "value",
	"name___": "value",
}`,
}, {
	in: `{
	"name": "value",
	"name______": "value",


	"name_": "value",
	"name___": "value"
	}`,
	want: `
{
	"name":       "value",
	"name______": "value",

	"name_":   "value",
	"name___": "value"
}`,
}, {
	in: `{
			/**/ "name": "value",
			/**/ "name______": "value",/**/
			"name_"/**/: "value"/**/,
			"name___":/**/ "value"
		}`,
	want: `
{
	/**/ "name":         "value",
	/**/ "name______":   "value", /**/
	"name_" /**/ :  "value" /**/ ,
	"name___": /**/ "value",
}`,
}, {
	in: `{"foo": "bar", 
	// Comment1
	"fizz":"buzz"
	// Comment2
,}`,
	want: `
{
	"foo": "bar",
	// Comment1
	"fizz": "buzz"
		// Comment2
		,
}`,
}, {
	in: `

	//ACls

// ACLs
{


	
					// foo
					// foo


					"k"


					// bar
					// bar


					:



					// baz
					// baz






	
								[									"v",									]

								// gaz
								// gaz

		     // ,

			  // maz




}
	`,
	want: `
//ACls

// ACLs
{
	// foo
	// foo

	"k"
		// bar
		// bar
		:
		// baz
		// baz
		["v"],

	// gaz
	// gaz

	// ,

	// maz
}`,
}, {
	in: `			   {
		"a" :     {	
			"b" : [
  
			  ],
		},
  
  
  
  }   `,
	want: `
{
	"a": {
		"b": [],
	},
}`,
}, {
	in: `{"a":{"b":[],"c":[





	],},}`,
	want: `{"a": {"b": [], "c": []}}`,
}, {
	in: `
	[
		[
			"a",
		]
		,
		[
			"a",
		]
		,
		[
			"a",
		]

	]
	
	`,
	want: `
[
	[
		"a",
	],
	[
		"a",
	],
	[
		"a",
	],
]`,
}, {
	in: `
	{//fizzbuzz
	"key"
		
		:"value"
		
		,//wizzwuzzz 
		
	// standalone comment
	
		// key comment
		"key":"value"}
	
	`,
	want: `
{ //fizzbuzz
	"key": "value", //wizzwuzzz

	// standalone comment

	// key comment
	"key": "value",
}`,
}}

func TestFormat(t *testing.T) {
	for _, tt := range testdataFormat {
		t.Run("", func(t *testing.T) {
			v, err := Parse([]byte(tt.in))
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			v.Format()
			got := v.String()
			want := strings.TrimPrefix(tt.want, "\n") + "\n"
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Format mismatch (-want +got):\n%s\n\ngot:\n%s\n\nwant:\n%s", diff, got, want)
			}
		})
	}
}
