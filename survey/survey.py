#!/usr/bin/python

"""Questionnaire app."""

import argparse
import json
import uuid
import datetime
import os
import hashlib
import time
import random
import string
import threading
import urllib.parse
import logging
import logging.handlers
import survey.version as ver
from flask import Flask, redirect, render_template, url_for, request, jsonify
app = Flask(__name__)

# where questions are stored and file naming for them
CONFIG_FILE_EXT = '.config'

# key, within app context, where question definitions are stored
QUESTION_KEY = 'questions'

# output method
METHOD_KEY = "method"
OUT_METHOD = "_out_method_"

# json fields to get values
Q_ID = 'q_id'
Q_TEXT = 'q_text'

# used in the locations we need to prevent multiple threads from interacting
LOCK = threading.RLock()

# Store the input question sets into the app context
app.config[QUESTION_KEY] = None

SNAPTIME_KEY = 'snapshot-time'
ADMIN_CODE = "admin-code"
TAG_KEY = "tag-key"
ARTIFACT_KEY = "artifact-key"

# JSON results
FAIL_JSON = "failed"

# Logging output
_LOG_FILE = "/var/log/epiphyte.survey.log"


@app.errorhandler(Exception)
def handle_exceptions(e):
    """Generic exception handler."""
    app.logger.error("an exception has been caught")
    app.logger.error(e)
    return "an error has been encountered"


def _get_config_path(index):
    """Retrieve the path to the config file."""
    questions_in = app.config[QUESTION_KEY][index]
    return questions_in


def _get_questions(index, defaults=None):
    """Get question set."""
    question_in = _get_config_path(index)
    with open(question_in, 'r') as f:
        conf = json.loads(f.read())

        meta = conf['meta']
        title = meta['title']
        anon = meta['anon']

        questions = conf['questions']
        question_set = []
        q_id = 0
        for question in questions:
            q_type = question['type']
            q_text = question['text']
            q_desc = question['desc']
            q_opts = []
            q_opt_key = "options"
            q_val = ""
            if q_opt_key in question:
                q_opts = question[q_opt_key]
            if defaults and q_text in defaults:
                q_val = defaults[q_text]
            question_set.append({'q_type': q_type,
                                 Q_TEXT: q_text,
                                 'q_desc': q_desc,
                                 'q_opts': q_opts,
                                 'q_val': q_val,
                                 Q_ID: str(q_id)})
            q_id = q_id + 1
        return (title, anon, question_set)


@app.route('/')
def home():
    """Home shows a simple 'begin' page."""
    query_params = _get_query_params()
    return render_template('begin.html', qparams=query_params)


def _get_query_params():
    """Get query parameters."""
    params = []
    if request.args:
        for item in request.args:
            val = request.args.get(item)
            params.append("{}={}".format(urllib.parse.quote(item),
                                         urllib.parse.quote(val)))
    query_params = ""
    if len(params) > 0:
        query_params = "?{}".format("&".join(params))
    return query_params


@app.route('/begin')
def begin():
    """Redirection wrapper to create the uuid for the session."""
    query_params = _get_query_params()
    return redirect(url_for('survey',
                            uuid=str(uuid.uuid4()),
                            idx=0) + query_params)


@app.route('/<int:idx>/<uuid>')
def survey(idx, uuid):
    """Survey started."""
    params = {}
    if request.args:
        for arg in request.args:
            params[arg] = request.args.get(arg)
    q = _get_questions(idx, params)
    do_follow = len(app.config[QUESTION_KEY]) > idx + 1
    follow = None
    if do_follow:
        follow = idx + 1
    return render_template('survey.html',
                           title=q[0],
                           anon=q[1],
                           questions=q[2],
                           session_id=uuid,
                           idx=idx,
                           do_follow=str(do_follow).lower(),
                           follow=follow,
                           snapshot_at=app.config[SNAPTIME_KEY],
                           qparams=_get_query_params())


@app.route("/<mode>/<int:idx>", methods=['POST'])
def snapshot(mode, idx):
    """Save a snapshot/submit of a survey."""
    return _save(idx, mode)


@app.route("/completed")
def completed():
    """Survey completed."""
    return render_template('complete.html', qparams=_get_query_params())


@app.route("/admin/<code>/<mode>")
def admin(code, mode):
    """Administrate the survey."""
    results = {FAIL_JSON: "unknown"}
    store = None
    with LOCK:
        store = app.config[ARTIFACT_KEY]
    if app.config[ADMIN_CODE] == code:
        if mode == "reload":
            exit(10)
        elif mode == "shutdown":
            exit(0)
        elif mode == "results":
            with LOCK:
                files = [f for f in
                         os.listdir(store)
                         if os.path.isfile(os.path.join(store, f))]
                files = [f for f in files if f.startswith(app.config[TAG_KEY])]
                results = files
        else:
            with LOCK:
                artifact_obj = os.path.join(store, mode)
                if os.path.exists(artifact_obj):
                    with open(artifact_obj) as f:
                        results = json.loads(f.read())
                else:
                    results[FAIL_JSON] = "command? {}".format(mode)
    else:
        results[FAIL_JSON] = "code? {}".format(code)
    return jsonify(results)


def _clean(value):
    """Clean invalid path chars from variables."""
    return "".join(c for c in value if c.isalnum() or c == '-')


def _save(idx, method):
    """Save/snapshot a survey input."""
    q = _get_questions(idx)[2]
    results = {}
    now_time = datetime.datetime.now()
    use_time = str(now_time)
    use_client = request.remote_addr
    results['time'] = use_time
    results['client'] = use_client
    session = "none"
    for key in request.form:
        val = request.form[key]
        use_key = key
        for item in q:
            if key == item[Q_ID]:
                use_key = item[Q_TEXT]
        results[use_key] = val
        if key == "session":
            session = val

    questions_in = _get_config_path(idx)
    use_method = app.config[METHOD_KEY]
    save_obj = SaveObject(results,
                          use_client,
                          session,
                          method,
                          questions_in)
    use_method(save_obj)
    return ""


class SaveObject(object):
    """save object."""

    def __init__(self,
                 results,
                 use_client,
                 session,
                 method,
                 questions_in):
        """object init."""
        self.results = results
        self.use_client = use_client
        self.session = session
        self.method = method
        self.questions_in = questions_in


def _out_method_off(obj):
    """for demo purposes."""
    pass


def _create_simple_id():
    """Create a simple id."""
    return ''.join(random.choices(string.ascii_uppercase + string.digits, k=6))


def _out_method_disk(obj):
    """disk storage."""
    dir_name = _build_output_path()
    parts = []
    for item in [obj.use_client,
                 _clean(_create_simple_id()),
                 obj.session]:
        parts.append(_clean(item))
    unique_name = "_".join(parts)
    time_id = _clean(str(time.time()))
    while len(time_id) < 20:
        time_id = time_id + "0"
    out_name = "{0}_{1}_{2}_{3}".format(app.config[TAG_KEY],
                                        time_id,
                                        _clean(obj.method)[0:4],
                                        unique_name)
    with open(dir_name + out_name, 'w') as f:
        f.write(json.dumps(obj.results,
                           sort_keys=True,
                           indent=4,
                           separators=(',', ': ')))


def _build_output_path():
    """build an output path."""
    base_dir = None
    with LOCK:
        base_dir = app.config[ARTIFACT_KEY]
        if not os.path.exists(base_dir):
            os.makedirs(base_dir)
    return base_dir + "/"


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(description='Survey')
    parser.add_argument('--host', type=str, default="0.0.0.0",
                        help='host name')
    parser.add_argument('--snapshot', type=int, default=15,
                        help='auto snapshot (<= 0 is disabled)')
    parser.add_argument('--port', type=int, default=8080,
                        help='port to operate on')
    parser.add_argument('--questions', nargs='+', type=str,
                        help='a json file expressing questions')
    methods = [x.replace(OUT_METHOD,
                         "") for x in dir() if x.startswith(OUT_METHOD)]
    parser.add_argument('--output', default="disk",
                        choices=methods,
                        help="output method")
    parser.add_argument('--code', default='running', help='admin url code')
    now = datetime.datetime.now().isoformat().replace(":", "-")
    now = now[0:19]
    parser.add_argument('--tag', default=now, help="output tag")
    parser.add_argument('--store',
                        default="/var/db/survey/",
                        help="data store")
    parser.add_argument('--config',
                        default="/etc/survey/",
                        help="survey config files")
    args = parser.parse_args()
    app.config[QUESTION_KEY] = []
    app.config[ARTIFACT_KEY] = args.store
    app.config[TAG_KEY] = _clean(args.tag)
    app.config[METHOD_KEY] = globals()[OUT_METHOD + args.output]
    app.config[SNAPTIME_KEY] = args.snapshot
    app.config[ADMIN_CODE] = args.code
    if args.questions is None or len(args.questions) == 0:
        print('question set(s) required')
        exit(1)
    for q in args.questions:
        set_questions = os.path.join(args.config, q + CONFIG_FILE_EXT)
        if not os.path.exists(set_questions):
            print("{0} does not exist...".format(set_questions))
            exit(-1)
        app.config[QUESTION_KEY].append(set_questions)
    print("survey ({})".format(ver.__version__))
    print("tag: {}".format(args.tag))
    handler = logging.handlers.RotatingFileHandler(_LOG_FILE,
                                                   maxBytes=10000,
                                                   backupCount=10)
    handler.setLevel(logging.INFO)
    formatter = logging.Formatter('%(asctime)s - %(levelname)s - %(message)s')
    handler.setFormatter(formatter)
    app.logger.addHandler(handler)
    app.run(host=args.host, port=args.port)
    exit(0)


if __name__ == "__main__":
    main()
