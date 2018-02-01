const electron = require('electron')
const app = electron.app
const BrowserWindow = electron.BrowserWindow
const path = require('path')
const fs = require('fs')
import { session } from 'electron';
let mainWindow
function createWindow () {
  session.defaultSession.webRequest.onBeforeSendHeaders((details, callback) => {
    details.requestHeaders['User-Agent'] = 'electron-survey';
    callback({ cancel: false, requestHeaders: details.requestHeaders });
  });
  let config = path.join(app.getPath('userData'), 'survey.txt')
  let url = undefined
  if (fs.existsSync(config)) {
      url = fs.readFileSync(config, 'utf8').replace(/[^\x00-\x7F]/g, "")
  }
  if (!url || url.length === 0 || url === undefined) {
    url = "http://localhost:8080"
  }
  mainWindow = new BrowserWindow({width: 1024, height: 768})
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

