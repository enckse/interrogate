#!/usr/bin/python

"""Questionnaire app."""

import argparse
import json
import uuid
import datetime
import os
import hashlib
import time
import threading
from flask import Flask, redirect, render_template, url_for, request, jsonify
app = Flask(__name__)

# where questions are stored and file naming for them
QUESTION_DIR = 'questions'
CONFIG_FILE_EXT = '.config'

# key, within app context, where question definitions are stored
QUESTION_KEY = 'questions'

# output method
METHOD_KEY = "method"
OUT_METHOD = "_out_method_"
ARTIFACTS = "artifacts"

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

def _get_config_path(index):
    """Retrieve the path to the config file."""
    questions_in = app.config[QUESTION_KEY][index]
    return questions_in


def _get_questions(index):
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
            if q_opt_key in question:
                q_opts = question[q_opt_key]

            question_set.append({'q_type': q_type,
                                 Q_TEXT: q_text,
                                 'q_desc': q_desc,
                                 'q_opts': q_opts,
                                 Q_ID: str(q_id)})
            q_id = q_id + 1
        return (title, anon, question_set)


@app.route('/')
def home():
    """Home shows a simple 'begin' page."""
    return render_template('begin.html')


@app.route('/begin')
def begin():
    """Redirection wrapper to create the uuid for the session."""
    return redirect(url_for('survey', uuid=str(uuid.uuid4()), idx=0))


@app.route('/<int:idx>/<uuid>')
def survey(idx, uuid):
    """Survey started."""
    q = _get_questions(idx)
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
                           snapshot_at=app.config[SNAPTIME_KEY])


@app.route("/<mode>/<int:idx>", methods=['POST'])
def snapshot(mode, idx):
    """Save a snapshot/submit of a survey."""
    return _save(idx, mode)


@app.route("/completed")
def completed():
    """Survey completed."""
    return render_template('complete.html')

@app.route("/admin/<code>/<mode>")
def admin(code, mode):
    """Administrate the survey."""
    results = ""
    if app.config[ADMIN_CODE] == code:
        if mode == "reload":
            exit(10)
        elif mode == "shutdown":
            exit(0)
        elif mode == "results":
            files = [f for f in 
                     os.listdir(ARTIFACTS)
                     if os.path.isfile(os.path.join(ARTIFACTS, f))]
            files = [f for f in files if f.startswith(app.config[TAG_KEY])]
            results = files
        else:
            print("unknown command: {}".format(mode))
    else:
        print("invalid code: {}".format(code))
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
    today = now_time.strftime("%Y-%m-%d")
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
    config_name = os.path.split(questions_in)[-1].replace(CONFIG_FILE_EXT, "")
    use_method = app.config[METHOD_KEY]
    save_obj = SaveObject(results,
                          today,
                          config_name,
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
                 today,
                 config_name,
                 use_client,
                 session,
                 method,
                 questions_in):
        """object init."""
        self.results = results
        self.today = today
        self.config_name = config_name
        self.use_client = use_client
        self.session = session
        self.method = method
        self.questions_in = questions_in


def _out_method_off(obj):
    """for demo purposes."""
    pass


def _out_method_disk(obj):
    """disk storage."""
    dir_name = _build_output_path()
    parts = []
    for item in [obj.today,
                 obj.config_name,
                 obj.use_client,
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
    base_dir = ARTIFACTS
    with LOCK:
        if not os.path.exists(base_dir):
            os.makedirs(base_dir)
    return base_dir + "/"

if __name__ == "__main__":
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
    parser.add_argument('--tag', default=now, help="output tag")
    args = parser.parse_args()
    app.config[QUESTION_KEY] = []
    app.config[TAG_KEY] = _clean(args.tag)
    app.config[METHOD_KEY] = globals()[OUT_METHOD + args.output]
    app.config[SNAPTIME_KEY] = args.snapshot
    app.config[ADMIN_CODE] = args.code
    if args.questions is None or len(args.questions) == 0:
        print('question set(s) required')
        exit(1)
    for q in args.questions:
        set_questions = os.path.join(QUESTION_DIR, q + CONFIG_FILE_EXT)
        if not os.path.exists(set_questions):
            print("{0} does not exist...".format(set_questions))
            exit(-1)
        app.config[QUESTION_KEY].append(set_questions)
    print("survey (__VERSION__)")
    app.run(host=args.host, port=args.port)
    exit(0)
