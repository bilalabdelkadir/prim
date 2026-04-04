export interface Model {
  name: string;
  table_name: string;
}

export interface Field {
  name: string;
  type: string;
  column_name: string;
  is_optional: boolean;
  is_primary: boolean;
  is_unique: boolean;
  default_value: string;
  attributes: string[];
}

export interface Relation {
  name: string;
  type: string;
  model: string;
  foreign_key: string;
  references: string;
  is_array?: boolean;
}

export interface PrimQuery {
  name: string;
  model: string;
  operation: 'findOne' | 'findMany' | 'count';
  select: string[];
  where: WhereCondition[];
  orderBy: OrderByDef[];
  limit: number | null;
  skip: number | null;
  include: IncludeNode[];
}

export interface IncludeNode {
  id: string;
  relationName: string;
  modelName: string;
  isArray: boolean;
  foreignKey: string;
  referenceKey: string;
  select: string[];
  where: WhereCondition[];
  orderBy: OrderByDef[];
  limit: number | null;
  include: IncludeNode[];
  collapsed: boolean;
}

export interface WhereCondition {
  id: string;
  field: string;
  operator: string;
  paramName: string;
  paramType: string;
}

export interface JoinDef {
  id: string;
  model: string;
  joinType: 'INNER' | 'LEFT';
  fields: string[];
}

export interface OrderByDef {
  id: string;
  field: string;
  direction: 'ASC' | 'DESC';
}

export interface QueryDefinition {
  model: string;
  operation: 'findOne' | 'findMany' | 'count';
  methodName: string;
  fields: string[];
  where: Array<{
    field: string;
    operator: string;
    paramName: string;
    paramType: string;
  }>;
  joins: Array<{
    model: string;
    joinType: string;
    fields: string[];
  }>;
  orderBy: Array<{
    field: string;
    direction: string;
  }>;
  limit: number | null;
}

export type View = 'tables' | 'query' | 'sql';
