import { useState, useEffect, useMemo } from 'react'
import type { Model, Field, Relation, PrimQuery, IncludeNode, WhereCondition, OrderByDef, DataField } from '../types'
import IncludeNodeEditor from './IncludeNodeEditor'
import LiveCodePreview from './LiveCodePreview'

interface QueryBuilderProps {
  models: Model[];
}

const OPERATORS = ['=', '!=', '>', '<', '>=', '<=', 'LIKE', 'IN', 'IS NULL'] as const
const PARAM_TYPES = ['string', 'int', 'bool', 'float64', 'time.Time'] as const

let nextId = 0
const genId = () => 'n' + nextId++

/* -- Collapsible Section ---------------------------------------- */

function CollapsibleSection({
  title,
  summary,
  defaultOpen = false,
  children,
}: {
  title: string;
  summary: string;
  defaultOpen?: boolean;
  children: React.ReactNode;
}) {
  const [open, setOpen] = useState(defaultOpen)

  return (
    <div className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 w-full text-left py-2 ui-text-label text-white/40 cursor-pointer hover:text-white/60 transition-all duration-150"
      >
        <span
          className="text-white/25 transition-transform duration-200 inline-block ui-text-xs"
          style={{ transform: open ? 'rotate(90deg)' : 'rotate(0deg)' }}
        >
          &#9656;
        </span>
        <span>{title}</span>
        {!open && summary && (
          <span className="text-white/20 ml-1">({summary})</span>
        )}
      </button>
      {open && (
        <div className="relative pb-3 space-y-2 border-l-2 border-white/[0.06] ml-1.5 pl-4">
          {/* Horizontal tick connecting to vertical line */}
          <div className="absolute top-0 left-0 w-2.5 border-t-2 border-white/[0.06]" />
          {children}
        </div>
      )}
    </div>
  )
}

/* -- QueryBuilder ----------------------------------------------- */

function QueryBuilder({ models }: QueryBuilderProps) {
  const [query, setQuery] = useState<PrimQuery>({
    name: '',
    model: '',
    operation: 'findMany',
    select: [],
    where: [],
    orderBy: [],
    limit: null,
    skip: null,
    include: [],
    data: [],
  })

  const [fields, setFields] = useState<Field[]>([])
  const [relations, setRelations] = useState<Relation[]>([])
  const [fieldsLoading, setFieldsLoading] = useState(false)
  const [showRelationPicker, setShowRelationPicker] = useState(false)
  const [saveLoading, setSaveLoading] = useState(false)
  const [toast, setToast] = useState<{ type: 'success' | 'error'; message: string } | null>(null)

  // Set initial model
  useEffect(() => {
    if (models.length > 0 && !query.model && models[0]) {
      setQuery(q => ({ ...q, model: models[0]!.name }))
    }
  }, [models, query.model])

  // Fetch fields + relations when model changes
  useEffect(() => {
    if (!query.model) {
      setFields([])
      setRelations([])
      return
    }

    setFieldsLoading(true)
    setQuery(q => ({ ...q, select: [], where: [], orderBy: [], include: [], limit: null, skip: null }))

    Promise.all([
      fetch(`/api/models/${query.model}/fields`).then(r => r.json()),
      fetch(`/api/models/${query.model}/relations`).then(r => r.json()),
    ])
      .then(([fieldsData, relationsData]: [Field[], Relation[]]) => {
        setFields(fieldsData)
        setRelations(relationsData ?? [])
        setQuery(q => ({ ...q, select: fieldsData.map(f => f.name) }))
      })
      .catch(() => {
        setFields([])
        setRelations([])
      })
      .finally(() => setFieldsLoading(false))
  }, [query.model])

  // Auto-dismiss toast
  useEffect(() => {
    if (!toast) return
    const t = setTimeout(() => setToast(null), 3000)
    return () => clearTimeout(t)
  }, [toast])

  // -- Field selection --
  const toggleField = (name: string) => {
    setQuery(q => ({
      ...q,
      select: q.select.includes(name) ? q.select.filter(f => f !== name) : [...q.select, name],
    }))
  }

  const toggleAllFields = () => {
    setQuery(q => ({
      ...q,
      select: q.select.length === fields.length ? [] : fields.map(f => f.name),
    }))
  }

  // -- Where conditions --
  const addCondition = () => {
    const firstField = fields[0]?.name ?? ''
    const newCond: WhereCondition = {
      id: genId(),
      field: firstField,
      operator: '=',
      paramName: '',
      paramType: 'string',
    }
    setQuery(q => ({ ...q, where: [...q.where, newCond] }))
  }

  const updateCondition = (id: string, patch: Partial<WhereCondition>) => {
    setQuery(q => ({
      ...q,
      where: q.where.map(c => (c.id === id ? { ...c, ...patch } : c)),
    }))
  }

  const removeCondition = (id: string) => {
    setQuery(q => ({ ...q, where: q.where.filter(c => c.id !== id) }))
  }

  // -- Order By --
  const addOrderBy = () => {
    const firstField = fields[0]?.name ?? ''
    const newOb: OrderByDef = { id: genId(), field: firstField, direction: 'ASC' }
    setQuery(q => ({ ...q, orderBy: [...q.orderBy, newOb] }))
  }

  const updateOrderBy = (id: string, patch: Partial<OrderByDef>) => {
    setQuery(q => ({
      ...q,
      orderBy: q.orderBy.map(o => (o.id === id ? { ...o, ...patch } : o)),
    }))
  }

  const removeOrderBy = (id: string) => {
    setQuery(q => ({ ...q, orderBy: q.orderBy.filter(o => o.id !== id) }))
  }

  // -- Data Fields (for create/update) --
  const addDataField = () => {
    const firstField = scalarFields[0]?.name ?? ''
    const newDf: DataField = { id: genId(), fieldName: firstField, paramName: firstField, paramType: 'string' }
    setQuery(q => ({ ...q, data: [...q.data, newDf] }))
  }

  const updateDataField = (id: string, patch: Partial<DataField>) => {
    setQuery(q => ({
      ...q,
      data: q.data.map(d => (d.id === id ? { ...d, ...patch } : d)),
    }))
  }

  const removeDataField = (id: string) => {
    setQuery(q => ({ ...q, data: q.data.filter(d => d.id !== id) }))
  }

  const scalarFields = fields.filter(f => !f.type.match(/^[A-Z]/) || ['Int', 'String', 'Boolean', 'Float', 'DateTime'].includes(f.type))

  // -- Includes --
  const addInclude = (rel: Relation) => {
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
    setQuery(q => ({ ...q, include: [...q.include, child] }))
    setShowRelationPicker(false)
  }

  const updateInclude = (index: number, updated: IncludeNode) => {
    setQuery(q => {
      const newIncludes = [...q.include]
      newIncludes[index] = updated
      return { ...q, include: newIncludes }
    })
  }

  const removeInclude = (index: number) => {
    setQuery(q => ({ ...q, include: q.include.filter((_, i) => i !== index) }))
  }

  // -- Save --
  const handleSave = () => {
    const modelName = query.model.toLowerCase()
    const outputPath = `generated/${modelName}_repository.go`
    setSaveLoading(true)
    fetch('/api/query/build/save', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...query, outputPath }),
    })
      .then(r => {
        if (!r.ok) return r.text().then(t => { throw new Error(t) })
        return r.json()
      })
      .then((data: { message?: string }) => setToast({ type: 'success', message: data.message ?? 'Query saved!' }))
      .catch(err => setToast({ type: 'error', message: err.message || 'Failed to save' }))
      .finally(() => setSaveLoading(false))
  }

  const queryIsReady = query.model && query.name
  const liveQuery = useMemo(() => (queryIsReady ? query : null), [query, queryIsReady])

  const selectClass =
    'bg-white/[0.03] border border-white/[0.06] focus:border-white/30 ui-text-base font-mono text-white/80 rounded-sm px-2 py-1 outline-none cursor-pointer transition-all duration-150'
  const inputClass =
    'bg-white/[0.03] border border-white/[0.06] focus:border-white/30 ui-text-sm font-mono rounded-sm px-2 py-1 outline-none transition-all duration-150'

  const selectSummary =
    query.select.length === fields.length && fields.length > 0
      ? 'all'
      : `${query.select.length} selected`

  const whereSummary = query.where.length > 0 ? `${query.where.length} conditions` : 'none'
  const orderSummary = query.orderBy.length > 0 ? `${query.orderBy.length}` : 'none'
  const dataSummary = query.data.length > 0 ? `${query.data.length} fields` : 'none'

  const isRead = ['findOne', 'findMany', 'count'].includes(query.operation)
  const isWrite = ['create', 'update'].includes(query.operation)
  const isDelete = query.operation === 'delete'

  return (
    <div className="flex h-full">
      {/* -- Left Panel: Query Tree Builder -- */}
      <div className="flex-1 overflow-y-auto border-r border-white/10 p-4 space-y-3">
        {/* Header bar: method name, model, operation, save — all on one line */}
        <div className="flex items-center gap-3">
          <input
            type="text"
            value={query.name}
            onChange={e => setQuery(q => ({ ...q, name: e.target.value }))}
            placeholder="MethodName"
            className="flex-1 bg-transparent border-b border-white/15 focus:border-white/40 ui-text-lg font-mono text-white pb-1 outline-none transition-all duration-150 min-w-0"
          />
          <select
            value={query.model}
            onChange={e => setQuery(q => ({ ...q, model: e.target.value }))}
            className={`${selectClass} w-32`}
          >
            <option value="">Model...</option>
            {models.map(m => (
              <option key={m.name} value={m.name}>{m.name}</option>
            ))}
          </select>
          <div className="flex bg-white/[0.02] rounded-sm border border-white/[0.06] overflow-hidden">
            {([
              { key: 'findOne', label: 'One' },
              { key: 'findMany', label: 'Many' },
              { key: 'count', label: 'Count' },
              { key: 'create', label: 'Create' },
              { key: 'update', label: 'Update' },
              { key: 'delete', label: 'Delete' },
            ] as const).map(op => (
              <button
                key={op.key}
                onClick={() => setQuery(q => ({ ...q, operation: op.key as PrimQuery['operation'] }))}
                className={`ui-text-label px-2 py-1.5 cursor-pointer transition-all duration-150 ${
                  query.operation === op.key
                    ? op.key === 'delete' ? 'text-red-400 bg-red-400/10' : 'text-[#05df72] bg-[rgba(5,223,114,0.08)]'
                    : 'text-white/30 hover:text-white/60'
                }`}
              >
                {op.label}
              </button>
            ))}
          </div>
          <button
            onClick={handleSave}
            disabled={saveLoading || !query.name || !query.model}
            className="bg-[#05df72] text-black ui-text-label font-semibold px-3 py-1.5 rounded-sm hover:bg-[#00e87a] transition-all duration-150 cursor-pointer
              disabled:opacity-40 disabled:cursor-not-allowed whitespace-nowrap"
          >
            {saveLoading ? 'SAVING...' : 'SAVE'}
          </button>
        </div>

        {/* Root query card */}
        {query.model && (
          <div className="space-y-0">
            {fieldsLoading ? (
              <p className="ui-text-sm text-white/30 py-4">Loading fields...</p>
            ) : (
              <>
                {/* Data Fields — for create/update */}
                {isWrite && (
                  <CollapsibleSection title="Data Fields" summary={dataSummary} defaultOpen>
                    {query.data.length === 0 ? (
                      <p className="ui-text-xs text-white/20">No data fields. Add fields to set values.</p>
                    ) : (
                      <div className="space-y-1.5">
                        {query.data.map(df => (
                          <div key={df.id} className="flex gap-1.5 items-center">
                            <select
                              value={df.fieldName}
                              onChange={e => {
                                const name = e.target.value
                                updateDataField(df.id, { fieldName: name, paramName: name })
                              }}
                              className={`${inputClass} w-32 cursor-pointer`}
                            >
                              {scalarFields.map(f => (
                                <option key={f.name} value={f.name}>{f.name}</option>
                              ))}
                            </select>
                            <input
                              type="text"
                              value={df.paramName}
                              onChange={e => updateDataField(df.id, { paramName: e.target.value })}
                              placeholder="paramName"
                              className={`${inputClass} flex-1 min-w-0`}
                            />
                            <select
                              value={df.paramType}
                              onChange={e => updateDataField(df.id, { paramType: e.target.value })}
                              className={`${inputClass} w-24 cursor-pointer`}
                            >
                              {PARAM_TYPES.map(pt => (
                                <option key={pt} value={pt}>{pt}</option>
                              ))}
                            </select>
                            <button
                              onClick={() => removeDataField(df.id)}
                              className="text-white/20 hover:text-red-400 transition-all duration-150 cursor-pointer flex-shrink-0 ui-text-base"
                            >
                              &times;
                            </button>
                          </div>
                        ))}
                      </div>
                    )}
                    <button
                      onClick={addDataField}
                      className="ui-text-label text-white/20 hover:text-[#05df72] transition-all duration-150 cursor-pointer mt-1"
                    >
                      + Add Field
                    </button>
                  </CollapsibleSection>
                )}

                {/* Select Fields — for read operations */}
                {(isRead || isWrite) && (
                <CollapsibleSection title={isWrite ? "Returning Fields" : "Select Fields"} summary={selectSummary} defaultOpen={isRead}>
                  <div className="flex items-center justify-end mb-1">
                    <button
                      onClick={toggleAllFields}
                      className="ui-text-label text-white/20 hover:text-[#05df72] transition-all duration-150 cursor-pointer"
                    >
                      {query.select.length === fields.length ? 'Deselect All' : 'Select All'}
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
                          checked={query.select.includes(f.name)}
                          onChange={() => toggleField(f.name)}
                          className="accent-[#05df72] flex-shrink-0"
                        />
                        <span className="font-mono ui-text-sm text-white/60 truncate">{f.name}</span>
                        <span className="ui-text-label text-white/20 ml-auto flex-shrink-0">{f.type}</span>
                      </label>
                    ))}
                  </div>
                </CollapsibleSection>
                )}

                {/* Where Conditions — for all except count */}
                {!isDelete || query.where.length > 0 ? (
                <CollapsibleSection title="Where" summary={whereSummary} defaultOpen={isDelete || isWrite}>
                  {query.where.length === 0 ? (
                    <p className="ui-text-xs text-white/20">No conditions. All records returned.</p>
                  ) : (
                    <div className="space-y-1.5">
                      {query.where.map(cond => (
                        <div key={cond.id} className="flex gap-1.5 items-center">
                          <select
                            value={cond.field}
                            onChange={e => updateCondition(cond.id, { field: e.target.value })}
                            className={`${inputClass} w-32 cursor-pointer`}
                          >
                            {fields.map(f => (
                              <option key={f.name} value={f.name}>{f.name}</option>
                            ))}
                          </select>
                          <select
                            value={cond.operator}
                            onChange={e => updateCondition(cond.id, { operator: e.target.value })}
                            className={`${inputClass} w-24 cursor-pointer`}
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
                              placeholder="paramName"
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
                            className="text-white/20 hover:text-red-400 transition-all duration-150 cursor-pointer flex-shrink-0 ui-text-base"
                          >
                            &times;
                          </button>
                        </div>
                      ))}
                    </div>
                  )}
                  <button
                    onClick={addCondition}
                    className="ui-text-label text-white/20 hover:text-[#05df72] transition-all duration-150 cursor-pointer mt-1"
                  >
                    + Add Condition
                  </button>
                </CollapsibleSection>
                ) : null}

                {/* Order By + Limit — only for read operations */}
                {isRead && (
                <CollapsibleSection title="Order / Limit" summary={orderSummary}>
                  {query.orderBy.length === 0 && (
                    <p className="ui-text-xs text-white/20">No ordering specified.</p>
                  )}
                  {query.orderBy.length > 0 && (
                    <div className="space-y-1.5">
                      {query.orderBy.map(ob => (
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
                            className="text-white/20 hover:text-red-400 transition-all duration-150 cursor-pointer flex-shrink-0 ui-text-base"
                          >
                            &times;
                          </button>
                        </div>
                      ))}
                    </div>
                  )}
                  <div className="flex items-center gap-3 mt-1.5">
                    <button
                      onClick={addOrderBy}
                      className="ui-text-label text-white/20 hover:text-[#05df72] transition-all duration-150 cursor-pointer"
                    >
                      + Add Order
                    </button>
                    <div className="flex-1" />
                    <span className="ui-text-label text-white/30">Limit</span>
                    <input
                      type="number"
                      value={query.limit ?? ''}
                      onChange={e => {
                        const val = e.target.value ? parseInt(e.target.value, 10) : null
                        setQuery(q => ({ ...q, limit: val }))
                      }}
                      placeholder="--"
                      min={0}
                      className={`${inputClass} w-16 text-center`}
                    />
                  </div>
                </CollapsibleSection>
                )}

                {/* Includes — for read and create operations */}
                {!isDelete && (
                <div className="pt-2 space-y-2">
                  <span className="ui-text-label text-white/40">Includes</span>

                  {query.include.length > 0 && (
                    <div className="space-y-2">
                      {query.include.map((node, i) => (
                        <IncludeNodeEditor
                          key={node.id}
                          node={node}
                          depth={0}
                          onUpdate={(updated) => updateInclude(i, updated)}
                          onRemove={() => removeInclude(i)}
                        />
                      ))}
                    </div>
                  )}

                  {/* Add Include */}
                  <div className="relative">
                    <button
                      onClick={() => setShowRelationPicker(!showRelationPicker)}
                      disabled={relations.length === 0}
                      className="ui-text-label text-white/20 hover:text-[#05df72] transition-all duration-150 cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
                    >
                      + Add Include
                    </button>
                    {showRelationPicker && relations.length > 0 && (
                      <div className="absolute z-10 mt-1 left-0 bg-[#111] border border-white/10 rounded-sm shadow-xl py-1 min-w-[200px]">
                        {relations
                          .filter(r => !query.include.some(inc => inc.relationName === r.name))
                          .map(rel => (
                            <button
                              key={rel.name}
                              onClick={() => addInclude(rel)}
                              className="w-full text-left px-3 py-1.5 ui-text-sm hover:bg-white/[0.05] text-white/50 hover:text-white/80 transition-all duration-150 cursor-pointer"
                            >
                              <span className="font-mono">{rel.name}</span>
                              <span className="text-white/25 ml-2">
                                ({rel.model}{rel.is_array || rel.type === 'hasMany' ? '[]' : ''})
                              </span>
                            </button>
                          ))}
                        {relations.filter(r => !query.include.some(inc => inc.relationName === r.name)).length === 0 && (
                          <p className="px-3 py-1.5 ui-text-xs text-white/20">All relations included</p>
                        )}
                      </div>
                    )}
                  </div>
                </div>
                )}
              </>
            )}
          </div>
        )}
      </div>

      {/* -- Right Panel: Live Code Preview (fixed width) -- */}
      <div className="w-[380px] flex-shrink-0 overflow-y-auto bg-[#050505]">
        <LiveCodePreview query={liveQuery} />
      </div>

      {/* Toast — fixed bottom-right, auto-dismiss */}
      {toast && (
        <div
          className={`fixed bottom-4 right-4 px-4 py-2.5 rounded-sm ui-text-sm font-mono shadow-xl border transition-all duration-300 z-50 ${
            toast.type === 'success'
              ? 'bg-[rgba(5,223,114,0.1)] border-[rgba(5,223,114,0.2)] text-[#05df72]'
              : 'bg-red-500/10 border-red-500/20 text-red-400'
          }`}
        >
          {toast.message}
        </div>
      )}
    </div>
  )
}

export default QueryBuilder
