const handlebars = require('handlebars')
const fs = require('fs')
const yaml = require('js-yaml')
const camelcase = require('camelcase')
const path = require('path')
const opts = require('./params')

console.log(opts)
const obj = yaml.safeLoad(fs.readFileSync(opts.def), 'utf-8')

const convertObj = {}

function getGoType(type) {
  switch (type) {
    case 'num':
    case 'char':
      return 'string'
    case 'byte':
      return '[]byte'
    default:
      throw new Error(type)
  }
}

function getTypeFlag(type) {
  switch (type) {
    case 'num':
      return { isNum: true }
    case 'char':
      return { isChar: true }
    case 'byte':
      return { isByte: true }
    default:
      throw new Error(type)
  }
}
function getType(rawType) {
  const type = rawType.replace(/^[lL]*/, '')
  const varLen = rawType.length - type.length
  if (varLen == 0) {
    return {
      type: getGoType(type),
      isFix: true,
      ...getTypeFlag(type),
    }
  } else {
    return {
      type: getGoType(type),
      varLenByteCount: Math.ceil(varLen / 2),
      isFix: false,
      ...getTypeFlag(type),
    }
  }
}

function getByteLength(type, length) {
  const t = type.replace(/^[lL]*/, '')
  switch (t) {
    case 'num':
      return Math.ceil(length / 2)
    case 'char':
    case 'byte':
      return length
    default:
      throw new Error(type)
  }
}

function findFieldInfo(id, fields) {
  const field = fields.find(f => f && f.id == id)

  if (field) {
    return field
  } else {
    throw new Error('not found filed id ' + id)
  }
}

function specialProcId(str) {
  return str
    .split(' ')
    .map(s => (s == 'id' ? 'i d' : s))
    .join(' ')
}

convertObj.fields = obj.fields.map(f => ({
  id: f.id,
  name: camelcase(specialProcId(f.name), { pascalCase: true }),
  length: getByteLength(f.type, f.length),
  description: f.description,
  ...getType(f.type),
}))

convertObj.trans = obj.trans.map(t => ({
  name: camelcase(specialProcId(t.name), { pascalCase: true }),
  req: {
    className: camelcase(specialProcId(t.name) + ' request', { pascalCase: true }),
    requiredFields: t.req.required.map(id => findFieldInfo(id, convertObj.fields)),
    conditionFields: t.req.conditionRequired.map(id => findFieldInfo(id, convertObj.fields)),
    description: t.req.description,
  },
  res: {
    className: camelcase(specialProcId(t.name) + ' response', { pascalCase: true }),
    requiredFields: t.res.required.map(id => findFieldInfo(id, convertObj.fields)),
    conditionFields: t.res.conditionRequired.map(id => findFieldInfo(id, convertObj.fields)),
    description: t.res.description,
  },
}))

const funcInitTmpl = handlebars.compile(fs.readFileSync(path.join(opts.templateDir, 'func_init.hbs'), 'utf-8'))
const modelReadTmpl = handlebars.compile(fs.readFileSync(path.join(opts.templateDir, 'model.read.hbs'), 'utf-8'))
const modelWriteTmpl = handlebars.compile(fs.readFileSync(path.join(opts.templateDir, 'model.write.hbs'), 'utf-8'))
const modelsTmpl = handlebars.compile(fs.readFileSync(path.join(opts.templateDir, 'models.hbs'), 'utf-8'))

handlebars.registerPartial('modelRead', modelReadTmpl)
handlebars.registerPartial('modelWrite', modelWriteTmpl)

console.log(modelsTmpl(convertObj))
