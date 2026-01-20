// KotlinImports.g4 - Minimal grammar for Gazelle dependency extraction
// Focuses only on package declarations and imports, ignores everything else
// Pure Go target, no CGO required

grammar KotlinImports;

// =============================================================================
// Parser Rules
// =============================================================================

// Entry point - parse package and imports, ignore the rest
kotlinFile
    : NL* fileAnnotations? packageHeader? importList
    ;

// File-level annotations like @file:JvmName("Foo")
fileAnnotations
    : (fileAnnotation NL*)+
    ;

fileAnnotation
    : AT 'file' COLON annotation
    ;

annotation
    : (simpleIdentifier DOT)* simpleIdentifier (LPAREN annotationArgs? RPAREN)?
    ;

annotationArgs
    : ~(LPAREN | RPAREN | NL)* (LPAREN annotationArgs RPAREN ~(LPAREN | RPAREN | NL)*)*
    ;

// Package declaration
packageHeader
    : PACKAGE qualifiedName semi
    ;

// Import list
importList
    : (importHeader)*
    ;

importHeader
    : IMPORT qualifiedName (DOT MULT)? importAlias? semi
    ;

importAlias
    : AS simpleIdentifier
    ;

// Qualified name (e.g., com.example.foo.Bar)
qualifiedName
    : simpleIdentifier (DOT simpleIdentifier)*
    ;

// Simple identifier - handles keywords used as identifiers with backticks
simpleIdentifier
    : IDENTIFIER
    | BACKTICK_IDENTIFIER
    // Soft keywords that can be used as identifiers
    | FILE
    | IMPORT
    | PACKAGE
    | AS
    ;

// Statement terminator - newline or semicolon
semi
    : (NL | SEMICOLON)+
    | EOF
    ;

// =============================================================================
// Lexer Rules
// =============================================================================

// Keywords
PACKAGE     : 'package';
IMPORT      : 'import';
AS          : 'as';
FILE        : 'file';

// Symbols
DOT         : '.';
MULT        : '*';
COLON       : ':';
SEMICOLON   : ';';
AT          : '@';
LPAREN      : '(';
RPAREN      : ')';

// Identifiers
IDENTIFIER
    : Letter LetterOrDigit*
    ;

BACKTICK_IDENTIFIER
    : '`' ~[\r\n`]+ '`'
    ;

fragment Letter
    : [a-zA-Z_]
    ;

fragment LetterOrDigit
    : [a-zA-Z0-9_]
    ;

// Whitespace and comments
NL          : '\r'? '\n';

WS          : [ \t]+ -> skip;

LINE_COMMENT
    : '//' ~[\r\n]* -> skip
    ;

BLOCK_COMMENT
    : '/*' .*? '*/' -> skip
    ;

// Skip everything after imports - we don't need to parse the rest
// This is a catch-all that consumes any remaining content
SKIP_REST
    : . -> skip
    ;
