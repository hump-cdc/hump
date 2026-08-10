package funcs

import (
	"fmt"
	"os"
	"text/template"
)

// Funcs is the canonical FuncMap registered with every parsed template in
// this project. It mirrors the stdlib text/template builtins (so call sites
// can replace the package map wholesale) and adds the project-specific
// helpers defined in sibling files.
//
// Coverage gap vs. Helm/sprig: htpasswd, derivePassword, semver/semverCompare,
// and TOML helpers are intentionally not provided. They would each pull in
// new module dependencies (bcrypt, semver, toml) and serve niche use cases.
var Funcs = template.FuncMap{
	// stdlib text/template builtins
	"and":      tmplAnd,
	"call":     tmplCall,
	"eq":       tmplEq,
	"ge":       tmplGe,
	"gt":       tmplGt,
	"html":     template.HTMLEscaper,
	"index":    tmplIndex,
	"js":       template.JSEscaper,
	"le":       tmplLe,
	"len":      tmplLength,
	"lt":       tmplLt,
	"ne":       tmplNe,
	"not":      tmplNot,
	"or":       tmplOr,
	"print":    fmt.Sprint,
	"printf":   fmt.Sprintf,
	"println":  fmt.Sprintln,
	"slice":    tmplSlice,
	"urlquery": template.URLQueryEscaper,

	// flow control
	"coalesce": tmplCoalesce,
	"default":  tmplDefault,
	"empty":    tmplEmpty,
	"fail":     tmplFail,
	"required": tmplRequired,
	"ternary":  tmplTernary,

	// case conversions
	"camelcase":       tmplCamelCase,
	"capitalcase":     tmplCapitalCase,
	"constantcase":    tmplConstantCase,
	"dotcase":         tmplDotCase,
	"headercase":      tmplHeaderCase,
	"kebabcase":       tmplKebabCase,
	"lower":           tmplLowerCase,
	"lowerfirst":      tmplLowerFirst,
	"pascalcase":      tmplPascalCase,
	"pascalsnakecase": tmplPascalSnakeCase,
	"pathcase":        tmplPathCase,
	"snakecase":       tmplSnakeCase,
	"upper":           tmplUpperCase,
	"upperfirst":      tmplUpperFirst,

	// strings
	"abbrev":     tmplAbbrev,
	"abbrevboth": tmplAbbrevBoth,
	"cat":        tmplCat,
	"contains":   tmplContains,
	"hasprefix":  tmplHasPrefix,
	"hassuffix":  tmplHasSuffix,
	"indent":     tmplIndent,
	"initials":   tmplInitials,
	"nindent":    tmplNindent,
	"nospace":    tmplNoSpace,
	"plural":     tmplPlural,
	"quote":      tmplQuote,
	"repeat":     tmplRepeat,
	"replace":    tmplReplace,
	"shuffle":    tmplShuffle,
	"squote":     tmplSquote,
	"substr":     tmplSubstr,
	"swapcase":   tmplSwapCase,
	"trim":       tmplTrim,
	"trimall":    tmplTrimAll,
	"trimprefix": tmplTrimPrefix,
	"trimsuffix": tmplTrimSuffix,
	"trunc":      tmplTrunc,
	"wrap":       tmplWrap,
	"wrapwith":   tmplWrapWith,

	// random strings
	"randalpha":    tmplRandAlpha,
	"randalphanum": tmplRandAlphaNum,
	"randascii":    tmplRandAscii,
	"randnumeric":  tmplRandNumeric,

	// type conversions
	"atoi":          tmplAtoi,
	"float64":       tmplFloat64,
	"fromjson":      tmplFromJSON,
	"fromjsonarray": tmplFromJSONArray,
	"int":           tmplInt,
	"int64":         tmplInt64,
	"json":          tmplJSON,
	"todecimal":     tmplToDecimal,
	"tostring":      tmplToString,
	"tostrings":     tmplToStrings,

	// yaml
	"fromyaml":      tmplFromYAML,
	"fromyamlarray": tmplFromYAMLArray,
	"yaml":          tmplYAML,

	// regex
	"regexfind":              tmplRegexFind,
	"regexfindall":           tmplRegexFindAll,
	"regexmatch":             tmplRegexMatch,
	"regexreplaceall":        tmplRegexReplaceAll,
	"regexreplaceallliteral": tmplRegexReplaceAllLiteral,
	"regexsplit":             tmplRegexSplit,

	// encoding
	"b32dec": tmplB32Dec,
	"b32enc": tmplB32Enc,
	"b64dec": tmplB64Dec,
	"b64enc": tmplB64Enc,

	// crypto
	"adler32":     tmplAdler32Sum,
	"aes_decrypt": tmplDecryptAES,
	"aes_encrypt": tmplEncryptAES,
	"randbytes":   tmplRandBytes,
	"sha1":        tmplSHA1Sum,
	"sha256":      tmplSHA256Sum,

	// date / time
	"ago":            tmplAgo,
	"date":           tmplDate,
	"dateinzone":     tmplDateInZone,
	"datemodify":     tmplDateModify,
	"duration":       tmplDuration,
	"durationround":  tmplDurationRound,
	"htmldate":       tmplHTMLDate,
	"htmldateinzone": tmplHTMLDateInZone,
	"now":            tmplNow,
	"todate":         tmplToDate,
	"unixepoch":      tmplUnixEpoch,

	// dict / map
	"deepcopy":       tmplDeepCopy,
	"dict":           tmplDict,
	"dig":            tmplDig,
	"get":            tmplGet,
	"haskey":         tmplHasKey,
	"keys":           tmplKeys,
	"merge":          tmplMerge,
	"mergeoverwrite": tmplMergeOverwrite,
	"omit":           tmplOmit,
	"pick":           tmplPick,
	"pluck":          tmplPluck,
	"set":            tmplSet,
	"unset":          tmplUnset,
	"values":         tmplValues,

	// lists
	"append":    tmplAppend,
	"chunk":     tmplChunk,
	"compact":   tmplCompact,
	"concat":    tmplConcat,
	"first":     tmplFirst,
	"has":       tmplHas,
	"initial":   tmplInitial,
	"last":      tmplLast,
	"list":      tmplList,
	"prepend":   tmplPrepend,
	"rest":      tmplRest,
	"reverse":   tmplReverse,
	"seq":       tmplSeq,
	"uniq":      tmplUniq,
	"until":     tmplUntil,
	"untilstep": tmplUntilStep,
	"without":   tmplWithout,

	// integer math
	"add":  tmplAdd,
	"add1": tmplAdd1,
	"div":  tmplDiv,
	"max":  tmplMax,
	"min":  tmplMin,
	"mod":  tmplMod,
	"mul":  tmplMul,
	"sub":  tmplSub,

	// float math
	"add1f": tmplAdd1F,
	"addf":  tmplAddF,
	"ceil":  tmplCeil,
	"divf":  tmplDivF,
	"floor": tmplFloor,
	"maxf":  tmplMaxF,
	"minf":  tmplMinF,
	"mulf":  tmplMulF,
	"round": tmplRound,
	"subf":  tmplSubF,

	// filepath
	"base":  tmplBase,
	"clean": tmplClean,
	"dir":   tmplDir,
	"ext":   tmplExt,
	"isabs": tmplIsAbs,

	// url
	"urljoin":  tmplURLJoin,
	"urlparse": tmplURLParse,

	// uuid
	"uuidv4": tmplUUIDv4,

	// tracing
	"span_id":  tmplSpanID,
	"trace_id": tmplTraceID,

	// os / env
	"env":      os.Getenv,
	"hostname": os.Hostname,
}
