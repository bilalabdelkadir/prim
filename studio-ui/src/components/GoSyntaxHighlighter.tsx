const KEYWORDS = new Set([
  'func', 'return', 'if', 'else', 'for', 'range', 'switch', 'case', 'default',
  'var', 'const', 'type', 'struct', 'interface', 'map', 'make', 'append',
  'nil', 'true', 'false', 'defer', 'go', 'select', 'chan', 'break', 'continue',
  'fallthrough', 'package', 'import', 'len', 'cap', 'new', 'delete', 'copy',
])

const TYPES = new Set([
  'int', 'int8', 'int16', 'int32', 'int64', 'uint', 'uint8', 'uint16', 'uint32', 'uint64',
  'string', 'bool', 'float32', 'float64', 'byte', 'rune', 'error', 'any',
  'Context', 'DB', 'Rows', 'Row', 'Time', 'Duration',
])

const OPERATORS = new Set(['=', ':=', '==', '!=', '<', '>', '<=', '>=', '+', '-', '*', '/', '&', '|', '!', '%', '&&', '||', '...'])
const PUNCTUATION = new Set(['{', '}', '(', ')', '[', ']', ',', ';', ':', '.'])

type TokenType = 'keyword' | 'type' | 'string' | 'comment' | 'number' | 'funcName' | 'punctuation' | 'operator' | 'default'

interface Token {
  type: TokenType
  value: string
}

const COLORS: Record<TokenType, string> = {
  keyword: '#05df72',
  type: '#00d294',
  string: 'rgba(255,255,255,0.55)',
  comment: 'rgba(255,255,255,0.25)',
  number: 'rgba(255,255,255,0.75)',
  funcName: '#ededed',
  punctuation: 'rgba(255,255,255,0.3)',
  operator: 'rgba(255,255,255,0.45)',
  default: 'rgba(255,255,255,0.7)',
}

function tokenizeLine(line: string, inBlockComment: boolean): { tokens: Token[]; inBlockComment: boolean } {
  const tokens: Token[] = []
  let i = 0
  let blockComment = inBlockComment

  if (blockComment) {
    const end = line.indexOf('*/')
    if (end === -1) {
      tokens.push({ type: 'comment', value: line })
      return { tokens, inBlockComment: true }
    }
    tokens.push({ type: 'comment', value: line.slice(0, end + 2) })
    i = end + 2
    blockComment = false
  }

  while (i < line.length) {
    const ch = line[i] as string
    const rest = line.slice(i)

    // Line comment
    if (rest.startsWith('//')) {
      tokens.push({ type: 'comment', value: rest })
      break
    }

    // Block comment start
    if (rest.startsWith('/*')) {
      const end = line.indexOf('*/', i + 2)
      if (end === -1) {
        tokens.push({ type: 'comment', value: rest })
        return { tokens, inBlockComment: true }
      }
      tokens.push({ type: 'comment', value: line.slice(i, end + 2) })
      i = end + 2
      continue
    }

    // Double-quoted string
    if (ch === '"') {
      let j = i + 1
      while (j < line.length && line[j] !== '"') {
        if (line[j] === '\\') j++
        j++
      }
      tokens.push({ type: 'string', value: line.slice(i, j + 1) })
      i = j + 1
      continue
    }

    // Backtick string
    if (ch === '`') {
      let j = i + 1
      while (j < line.length && line[j] !== '`') j++
      tokens.push({ type: 'string', value: line.slice(i, j + 1) })
      i = j + 1
      continue
    }

    // Whitespace
    if (ch === ' ' || ch === '\t') {
      let j = i
      while (j < line.length && (line[j] === ' ' || line[j] === '\t')) j++
      tokens.push({ type: 'default', value: line.slice(i, j) })
      i = j
      continue
    }

    // Multi-char operators
    if (i + 1 < line.length) {
      const two = line.slice(i, i + 2)
      if (OPERATORS.has(two)) {
        tokens.push({ type: 'operator', value: two })
        i += 2
        continue
      }
    }
    if (rest.startsWith('...')) {
      tokens.push({ type: 'operator', value: '...' })
      i += 3
      continue
    }

    // Single-char operators
    if (OPERATORS.has(ch)) {
      tokens.push({ type: 'operator', value: ch })
      i++
      continue
    }

    // Punctuation
    if (PUNCTUATION.has(ch)) {
      tokens.push({ type: 'punctuation', value: ch })
      i++
      continue
    }

    // Numbers
    if (ch >= '0' && ch <= '9') {
      let j = i
      while (j < line.length && ((line[j]! >= '0' && line[j]! <= '9') || line[j] === '.')) j++
      tokens.push({ type: 'number', value: line.slice(i, j) })
      i = j
      continue
    }

    // Words (identifiers/keywords)
    if ((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch === '_') {
      let j = i
      while (j < line.length && ((line[j]! >= 'a' && line[j]! <= 'z') || (line[j]! >= 'A' && line[j]! <= 'Z') || (line[j]! >= '0' && line[j]! <= '9') || line[j] === '_')) j++
      const word = line.slice(i, j)

      if (KEYWORDS.has(word)) {
        tokens.push({ type: 'keyword', value: word })
      } else if (TYPES.has(word)) {
        tokens.push({ type: 'type', value: word })
      } else {
        // Check if followed by ( → function call
        let k = j
        while (k < line.length && line[k] === ' ') k++
        if (k < line.length && line[k] === '(') {
          if (word[0]! >= 'A' && word[0]! <= 'Z' && !word.includes('_')) {
            tokens.push({ type: 'type', value: word })
          } else {
            tokens.push({ type: 'funcName', value: word })
          }
        } else if (word[0]! >= 'A' && word[0]! <= 'Z') {
          // PascalCase without ( — likely a type name
          tokens.push({ type: 'type', value: word })
        } else {
          tokens.push({ type: 'default', value: word })
        }
      }
      i = j
      continue
    }

    // Anything else
    tokens.push({ type: 'default', value: ch as string })
    i++
  }

  return { tokens, inBlockComment: blockComment }
}

function GoSyntaxHighlighter({ code }: { code: string }) {
  const lines = code.split('\n')
  // Remove trailing empty line if present
  if (lines.length > 0 && lines[lines.length - 1] === '') {
    lines.pop()
  }

  const gutterWidth = String(lines.length).length

  let inBlockComment = false
  const allLineTokens: Token[][] = []
  for (const line of lines) {
    const result = tokenizeLine(line, inBlockComment)
    allLineTokens.push(result.tokens)
    inBlockComment = result.inBlockComment
  }

  return (
    <pre
      className="bg-[#050505] overflow-x-auto py-3 rounded-sm m-0"
      style={{ fontFamily: "'Geist Mono', 'SF Mono', monospace", fontSize: '13px', lineHeight: '1.7' }}
    >
      {allLineTokens.map((tokens, lineIdx) => (
        <div
          key={lineIdx}
          className="flex hover:bg-white/[0.02] transition-colors duration-75 px-1"
        >
          <span
            className="select-none text-right text-white/15 border-r border-white/[0.06] pr-3 mr-4 flex-shrink-0"
            style={{ width: `${gutterWidth + 1.5}ch`, fontSize: '11px' }}
          >
            {lineIdx + 1}
          </span>
          <span className="flex-1">
            {tokens.length === 0 ? '\n' : tokens.map((tok, j) => (
              <span
                key={j}
                style={{
                  color: COLORS[tok.type],
                  fontWeight: tok.type === 'funcName' ? 500 : undefined,
                  fontStyle: tok.type === 'comment' ? 'italic' : undefined,
                }}
              >
                {tok.value}
              </span>
            ))}
          </span>
        </div>
      ))}
    </pre>
  )
}

export default GoSyntaxHighlighter
