import { useState, useEffect } from 'react'
import type { Model, Field } from '../types'

interface TableBrowserProps {
  models: Model[];
  selectedModel: string;
  onSelectModel: (name: string) => void;
}

function TableBrowser({ models: _models, selectedModel, onSelectModel: _onSelectModel }: TableBrowserProps) {
  const [fields, setFields] = useState<Field[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!selectedModel) return
    setLoading(true)
    fetch(`/api/models/${selectedModel}/fields`)
      .then(res => res.json())
      .then((data: Field[]) => setFields(data))
      .catch(() => setFields([]))
      .finally(() => setLoading(false))
  }, [selectedModel])

  // Build Prisma-like type string: "String?", "Int @id @default(...)", etc.
  const formatType = (field: Field): { type: string; attrs: string } => {
    let typePart = field.type
    if (field.is_optional) typePart += '?'

    const attrParts: string[] = []
    if (field.is_primary) attrParts.push('@id')
    if (field.is_unique && !field.is_primary) attrParts.push('@unique')
    if (field.default_value) attrParts.push(`@default(${field.default_value})`)
    if (field.attributes) {
      for (const attr of field.attributes) {
        // Avoid duplicating attrs we already handle
        if (!attr.startsWith('@id') && !attr.startsWith('@unique') && !attr.startsWith('@default')) {
          attrParts.push(attr)
        }
      }
    }

    return { type: typePart, attrs: attrParts.join(' ') }
  }

  if (!selectedModel) {
    return (
      <div className="flex items-center justify-center h-full">
        <span className="text-white/20 ui-text-sm tracking-wider font-sans">Select a model to view its fields</span>
      </div>
    )
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <span className="text-white/20 ui-text-sm tracking-wider font-sans">Loading fields...</span>
      </div>
    )
  }

  return (
    <div className="h-full overflow-y-auto p-4">
      <div className="mb-3">
        <h2 className="ui-text-lg font-medium text-white font-sans">{selectedModel}</h2>
        <p className="ui-text-sm text-white/30 mt-0.5 font-sans">
          {fields.length} field{fields.length !== 1 ? 's' : ''}
        </p>
      </div>

      {fields.length === 0 ? (
        <div className="flex items-center justify-center py-12">
          <span className="text-white/20 ui-text-sm tracking-wider font-sans">No fields found</span>
        </div>
      ) : (
        <table className="w-full">
          <thead>
            <tr className="border-b border-dashed border-white/10">
              <th className="text-left px-3 py-2 ui-text-label text-white/30 sticky top-0 bg-[#0a0a0a] z-10 font-sans">
                Field
              </th>
              <th className="text-left px-3 py-2 ui-text-label text-white/30 sticky top-0 bg-[#0a0a0a] z-10 font-sans">
                Type
              </th>
            </tr>
          </thead>
          <tbody>
            {fields.map((field, i) => {
              const { type, attrs } = formatType(field)
              return (
                <tr
                  key={field.name}
                  className={`border-b border-white/[0.04] hover:bg-white/[0.02] transition-all duration-150 ${
                    i === fields.length - 1 ? 'border-b-0' : ''
                  }`}
                >
                  <td className="px-3 py-2">
                    <span className="font-mono ui-text-base text-white/80">{field.name}</span>
                  </td>
                  <td className="px-3 py-2">
                    <span className="font-mono ui-text-sm text-[#05df72]/70">{type}</span>
                    {attrs && (
                      <span className="font-mono ui-text-sm text-white/30 ml-1.5">{attrs}</span>
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      )}
    </div>
  )
}

export default TableBrowser
