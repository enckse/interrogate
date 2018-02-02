const electron = require('electron')
const app = electron.app
const BrowserWindow = electron.BrowserWindow
const path = require('path')
const fs = require('fs')
let mainWindow
function createWindow () {
  let config = path.join(app.getPath('userData'), 'survey.txt')
  let url = undefined
  if (fs.existsSync(config)) {
      url = fs.readFileSync(config, 'utf8').replace(/[^\x00-\x7F]/g, "")
  }
  if (!url || url.length === 0 || url === undefined) {
    url = "http://localhost:8080"
  }

  let d = new Date()
  fs.appendFile(config + ".version", d.toString() + " -> " + app.getVersion() + "\n", function (e) {})
  mainWindow = new BrowserWindow({width: 1024, height: 768})
  mainWindow.loadURL(url, { userAgent: "electron-survey" })
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

