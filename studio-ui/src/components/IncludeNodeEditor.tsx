import { useState, useEffect } from 'react'
import type { IncludeNode, Field, Relation, WhereCondition, OrderByDef } from '../types'

interface IncludeNodeEditorProps {
  node: IncludeNode;
  depth: number;
  onUpdate: (updated: IncludeNode) => void;
  onRemove: () => void;
}

const DEPTH_COLORS = ['#05df72', '#00d294', 'rgba(255,255,255,0.3)', 'rgba(255,255,255,0.15)']
const OPERATORS = ['=', '!=', '>', '<', '>=', '<=', 'LIKE', 'IN', 'IS NULL'] as const
const PARAM_TYPES = ['string', 'int', 'bool', 'float64', 'time.Time'] as const

let nextIncludeId = 0
const genId = () => 'n' + nextIncludeId++

function IncludeNodeEditor({ node, depth, onUpdate, onRemove }: IncludeNodeEditorProps) {
  const [fields, setFields] = useState<Field[]>([])
  const [relations, setRelations] = useState<Relation[]>([])
  const [showRelationPicker, setShowRelationPicker] = useState(false)

  const borderColor = DEPTH_COLORS[depth % DEPTH_COLORS.length]

  const inputClass =
    'bg-white/[0.03] border border-white/[0.06] focus:border-white/30 ui-text-sm font-mono rounded-sm px-2 py-1 outline-none transition-all duration-150'

  useEffect(() => {
    if (!node.modelName) return
    Promise.all([
      fetch(`/api/models/${node.modelName}/fields`).then(r => r.json()),
      fetch(`/api/models/${node.modelName}/relations`).then(r => r.json()),
    ])
      .then(([fieldsData, relationsData]: [Field[], Relation[]]) => {
        setFields(fieldsData)
        setRelations(relationsData ?? [])
        // Auto-select all fields if none selected
        if (node.select.length === 0) {
          onUpdate({ ...node, select: fieldsData.map((f: Field) => f.name) })
        }
      })
      .catch(() => {
        setFields([])
        setRelations([])
      })
    // Only fetch on mount / model change
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [node.modelName])

  const toggleField = (name: string) => {
    const newSelect = node.select.includes(name)
      ? node.select.filter(f => f !== name)
      : [...node.select, name]
    onUpdate({ ...node, select: newSelect })
  }

  const toggleAllFields = () => {
    if (node.select.length === fields.length) {
      onUpdate({ ...node, select: [] })
    } else {
      onUpdate({ ...node, select: fields.map(f => f.name) })
    }
  }

  const addCondition = () => {
    const firstField = fields[0]?.name ?? ''
    const newCond: WhereCondition = {
      id: genId(),
      field: firstField,
      operator: '=',
      paramName: '',
      paramType: 'string',
    }
    onUpdate({ ...node, where: [...node.where, newCond] })
  }

  const updateCondition = (id: string, patch: Partial<WhereCondition>) => {
    onUpdate({
      ...node,
      where: node.where.map(c => (c.id === id ? { ...c, ...patch } : c)),
    })
  }

  const removeCondition = (id: string) => {
    onUpdate({ ...node, where: node.where.filter(c => c.id !== id) })
  }

  const addOrderBy = () => {
    const firstField = fields[0]?.name ?? ''
    const newOb: OrderByDef = { id: genId(), field: firstField, direction: 'ASC' }
    onUpdate({ ...node, orderBy: [...node.orderBy, newOb] })
  }

  const updateOrderBy = (id: string, patch: Partial<OrderByDef>) => {
    onUpdate({
      ...node,
      orderBy: node.orderBy.map(o => (o.id === id ? { ...o, ...patch } : o)),
    })
  }

  const removeOrderBy = (id: string) => {
    onUpdate({ ...node, orderBy: node.orderBy.filter(o => o.id !== id) })
  }

  const addChildInclude = (rel: Relation) => {
    const child: IncludeNode = {
      id: genId(),
      relationName: rel.name,
      modelName: rel.model,
      isArray: rel.is_array ?? rel.type === 'hasMany',
      foreignKey: rel.foreign_key,
      referenceKey: rel.references,
      select: [],
      where: [],
      orderBy: [],
      limit: null,
      include: [],
      collapsed: false,
    }
    onUpdate({ ...node, include: [...node.include, child] })
    setShowRelationPicker(false)
  }

  const updateChildInclude = (index: number, updated: IncludeNode) => {
    const newIncludes = [...node.include]
    newIncludes[index] = updated
    onUpdate({ ...node, include: newIncludes })
  }

  const removeChildInclude = (index: number) => {
    onUpdate({ ...node, include: node.include.filter((_, i) => i !== index) })
  }

  const toggleCollapsed = () => {
    onUpdate({ ...node, collapsed: !node.collapsed })
  }

  const selectSummary = node.select.length === fields.length && fields.length > 0
    ? 'all'
    : `${node.select.length}`

  return (
    <div
      className="border border-white/[0.06] rounded-sm bg-white/[0.015]"
      style={{ borderLeftWidth: '1.5px', borderLeftColor: borderColor }}
    >
      {/* Header — single compact line */}
      <div className="flex items-center gap-2 px-3 py-1.5">
        <button
          onClick={toggleCollapsed}
          className="text-white/25 hover:text-white/50 ui-text-xs transition-all duration-150 cursor-pointer"
        >
          <span
            className="inline-block transition-transform duration-200"
            style={{ transform: node.collapsed ? 'rotate(0deg)' : 'rotate(90deg)' }}
          >
            &#9656;
          </span>
        </button>
        <span className="font-mono ui-text-base text-white/80 font-medium">
          {node.relationName}
        </span>
        <span className="ui-text-label text-white/25 bg-white/[0.04] px-1.5 py-0.5 rounded-sm">
          {node.modelName}{node.isArray ? '[]' : ''}
        </span>
        {node.collapsed && (
          <span className="ui-text-label text-white/15 ml-1">{selectSummary} fields</span>
        )}
        <div className="flex-1" />
        <button
          onClick={onRemove}
          className="ui-text-label text-white/15 hover:text-red-400/80 cursor-pointer transition-all duration-150"
          title="Remove include"
        >
          REMOVE
        </button>
      </div>

      {/* Body */}
      {!node.collapsed && (
        <div className="px-3 pb-2 space-y-2">
          {/* Select Fields — inline label + grid, no collapsible wrapper */}
          <div>
            <div className="flex items-center justify-between mb-1">
              <span className="ui-text-label text-white/30">Fields</span>
              <button
                onClick={toggleAllFields}
                className="ui-text-label text-white/20 hover:text-[#05df72] transition-all duration-150 cursor-pointer"
              >
                {node.select.length === fields.length ? 'Deselect All' : 'Select All'}
              </button>
            </div>
            <div className="grid grid-cols-3 gap-1.5">
              {fields.map(f => (
                <label
                  key={f.name}
                  className="flex items-center gap-1.5 cursor-pointer transition-all duration-150 hover:bg-white/[0.02] rounded-sm px-1 py-0.5"
                >
                  <input
                    type="checkbox"
                    checked={node.select.includes(f.name)}
                    onChange={() => toggleField(f.name)}
                    className="accent-[#05df72] flex-shrink-0"
                  />
                  <span className="font-mono ui-text-sm text-white/60 truncate">{f.name}</span>
                  <span className="ui-text-label text-white/20 ml-auto flex-shrink-0">{f.type}</span>
                </label>
              ))}
            </div>
          </div>

          {/* Where Conditions — inline label */}
          {(node.where.length > 0) && (
            <div>
              <span className="ui-text-label text-white/30">Where</span>
              <div className="space-y-1.5 mt-1">
                {node.where.map(cond => (
                  <div key={cond.id} className="flex gap-1.5 items-center">
                    <select
                      value={cond.field}
                      onChange={e => updateCondition(cond.id, { field: e.target.value })}
                      className={`${inputClass} w-28 cursor-pointer`}
                    >
                      {fields.map(f => (
                        <option key={f.name} value={f.name}>{f.name}</option>
                      ))}
                    </select>
                    <select
                      value={cond.operator}
                      onChange={e => updateCondition(cond.id, { operator: e.target.value })}
                      className={`${inputClass} w-20 cursor-pointer`}
                    >
                      {OPERATORS.map(op => (
                        <option key={op} value={op}>{op}</option>
                      ))}
                    </select>
                    {cond.operator !== 'IS NULL' && (
                      <input
                        type="text"
                        value={cond.paramName}
                        onChange={e => updateCondition(cond.id, { paramName: e.target.value })}
                        placeholder="param"
                        className={`${inputClass} flex-1 min-w-0`}
                      />
                    )}
                    {cond.operator !== 'IS NULL' && (
                      <select
                        value={cond.paramType}
                        onChange={e => updateCondition(cond.id, { paramType: e.target.value })}
                        className={`${inputClass} w-24 cursor-pointer`}
                      >
                        {PARAM_TYPES.map(pt => (
                          <option key={pt} value={pt}>{pt}</option>
                        ))}
                      </select>
                    )}
                    <button
                      onClick={() => removeCondition(cond.id)}
                      className="text-white/20 hover:text-red-400 transition-all duration-150 cursor-pointer flex-shrink-0"
                    >
                      &times;
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Order By — inline */}
          {(node.orderBy.length > 0) && (
            <div>
              <span className="ui-text-label text-white/30">Order By</span>
              <div className="space-y-1.5 mt-1">
                {node.orderBy.map(ob => (
                  <div key={ob.id} className="flex gap-1.5 items-center">
                    <select
                      value={ob.field}
                      onChange={e => updateOrderBy(ob.id, { field: e.target.value })}
                      className={`${inputClass} flex-1 cursor-pointer`}
                    >
                      {fields.map(f => (
                        <option key={f.name} value={f.name}>{f.name}</option>
                      ))}
                    </select>
                    <div className="flex bg-white/[0.02] rounded-sm border border-white/[0.06] overflow-hidden">
                      {(['ASC', 'DESC'] as const).map(dir => (
                        <button
                          key={dir}
                          onClick={() => updateOrderBy(ob.id, { direction: dir })}
                          className={`ui-text-label px-2 py-1 cursor-pointer transition-all duration-150 ${
                            ob.direction === dir
                              ? 'text-[#05df72] bg-[rgba(5,223,114,0.08)]'
                              : 'text-white/30 hover:text-white/60'
                          }`}
                        >
                          {dir}
                        </button>
                      ))}
                    </div>
                    <button
                      onClick={() => removeOrderBy(ob.id)}
                      className="text-white/20 hover:text-red-400 transition-all duration-150 cursor-pointer"
                    >
                      &times;
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Action row: add condition, add order, limit */}
          <div className="flex items-center gap-3 pt-1">
            <button
              onClick={addCondition}
              className="ui-text-label text-white/20 hover:text-[#05df72] transition-all duration-150 cursor-pointer"
            >
              + Where
            </button>
            <button
              onClick={addOrderBy}
              className="ui-text-label text-white/20 hover:text-[#05df72] transition-all duration-150 cursor-pointer"
            >
              + Order
            </button>
            <div className="flex-1" />
            <span className="ui-text-label text-white/25">Limit</span>
            <input
              type="number"
              value={node.limit ?? ''}
              onChange={e => {
                const val = e.target.value ? parseInt(e.target.value, 10) : null
                onUpdate({ ...node, limit: val })
              }}
              placeholder="--"
              min={0}
              className={`${inputClass} w-14 text-center`}
            />
          </div>

          {/* Child Includes */}
          {node.include.length > 0 && (
            <div className="space-y-1.5 pt-1">
              {node.include.map((child, i) => (
                <IncludeNodeEditor
                  key={child.id}
                  node={child}
                  depth={depth + 1}
                  onUpdate={(updated) => updateChildInclude(i, updated)}
                  onRemove={() => removeChildInclude(i)}
                />
              ))}
            </div>
          )}

          {/* Add Include */}
          <div>
            {!showRelationPicker ? (
              <button
                onClick={() => setShowRelationPicker(true)}
                disabled={relations.length === 0}
                className="ui-text-label text-white/20 hover:text-[#05df72] transition-all duration-150 cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
              >
                + Include
              </button>
            ) : (
              <div className="bg-[#111] border border-white/10 rounded-sm py-1 mt-1">
                <div className="flex items-center justify-between px-3 py-1 border-b border-white/[0.06] mb-1">
                  <span className="ui-text-label text-white/30">Select relation</span>
                  <button
                    onClick={() => setShowRelationPicker(false)}
                    className="ui-text-label text-white/20 hover:text-white/50 cursor-pointer transition-colors"
                  >
                    &times;
                  </button>
                </div>
                {relations
                  .filter(r => !node.include.some(inc => inc.relationName === r.name))
                  .map(rel => (
                    <button
                      key={rel.name}
                      onClick={() => addChildInclude(rel)}
                      className="w-full text-left px-3 py-1.5 ui-text-sm hover:bg-white/[0.05] text-white/50 hover:text-white/80 transition-all duration-150 cursor-pointer"
                    >
                      <span className="font-mono">{rel.name}</span>
                      <span className="text-white/25 ml-1.5">
                        ({rel.model}{rel.is_array || rel.type === 'hasMany' ? '[]' : ''})
                      </span>
                    </button>
                  ))}
                {relations.filter(r => !node.include.some(inc => inc.relationName === r.name)).length === 0 && (
                  <p className="px-3 py-1.5 ui-text-xs text-white/20">All relations included</p>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

export default IncludeNodeEditor
