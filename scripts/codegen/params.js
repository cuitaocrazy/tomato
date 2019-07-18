const program = require('commander')

program
  .option('-d, --def <path>', '定义文件', 'fields_def.yaml')
  .option('-l, --lang <go>', '生成的语言，现在只有GO', 'go')
  .option('-t, --type <client|server|all>', '生成类型选项', 'server')
  .option('--template-dir <path>', '模板目录', 'templates')
  .option('-o, --output <path>', '输出目录', 'out')

program.parse(process.argv)

module.exports = program.opts()
