const handlebars = require('handlebars')
const fs = require('fs')
const yaml = require('js-yaml')
const camelcase = require('camelcase')

const obj = yaml.safeLoad(fs.readFileSync('./fields_def.yaml'), 'utf-8')

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
convertObj.fields = obj.fields.map(f => ({
  id: f.id,
  name: camelcase(f.name, { pascalCase: true }),
  length: getByteLength(f.type, f.length),
  description: f.description,
  ...getType(f.type),
}))

const template = handlebars.compile(fs.readFileSync('./templates/my.handlebars.go').toString())

fs.writeFileSync('./a.go', template(convertObj))
