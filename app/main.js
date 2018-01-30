const electron = require('electron')
const app = electron.app
const BrowserWindow = electron.BrowserWindow
const path = require('path')
const fs = require('fs')
let mainWindow
function createWindow () {
  let config = path.join(app.getPath('userData'), 'survey.url')
  let url = undefined
  if (fs.existsSync(config)) {
      url = fs.readFileSync(config)
  }
  if (!url || url.length === 0 || url === undefined) {
    url = "http://localhost:8080"
  }
  mainWindow = new BrowserWindow({width: 800, height: 600})
  mainWindow.loadURL(url)
  mainWindow.on('closed', function () {
    mainWindow = null
  })
}

app.on('ready', createWindow)

app.on('window-all-closed', function () {
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

app.on('activate', function () {
  if (mainWindow === null) {
    createWindow()
  }
})

