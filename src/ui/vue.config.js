const path = require('path')
module.exports = {
  configureWebpack: {
    resolve: {
      alias: {
        '@': path.resolve(__dirname, 'src'),
      }
    }
  },
  devServer: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  outputDir: 'dist',
  assetsDir: 'assets',
  publicPath: '/',
  chainWebpack: config => {
    config.plugin('html').tap(args => {
      args[0].template = path.resolve(__dirname, 'index.html')
      return args
    })
    if (config.plugins.has('copy')) {
      config.plugin('copy').tap(args => {
        args[0].patterns[0].globOptions = {
          ...args[0].patterns[0].globOptions,
          ignore: ['**/.*', '**/index.html']
        }
        return args
      })
    }
  }
}
