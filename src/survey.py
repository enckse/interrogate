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
CCACHE_KEY = "ccache"

# Logging output
_LOG_FILE = "/var/log/survey.log"


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
        question_idx = {}
        q_id = 0
        for question in questions:
            idx = q_id
            is_required = ""
            if "numbered" in question:
                idx = int(question["numbered"])
            if "attrs" in question:
                attrs = question["attrs"]
                if "required" in attrs:
                    is_required = "required"
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
            obj = {'q_type': q_type,
                   Q_TEXT: q_text,
                   'q_desc': q_desc,
                   'q_opts': q_opts,
                   'q_val': q_val,
                   Q_ID: str(q_id),
                   'q_req': is_required}
            if idx in question_idx:
                raise Exception("duplicate question index")
            question_idx[idx] = obj
            q_id = q_id + 1
        question_set = []
        for item in sorted(question_idx.keys()):
            question_set.append(question_idx[item])
        return (title, anon, question_set)


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
    do_c_cache = False
    c_cached = app.config[CCACHE_KEY]
    if c_cached is not None and len(c_cached) > 0:
        do_c_cache = True
    else:
        c_cached = ""
    return render_template('survey.html',
                           title=q[0],
                           anon=q[1],
                           questions=q[2],
                           session_id=uuid,
                           idx=idx,
                           do_follow=str(do_follow).lower(),
                           follow=follow,
                           c_cache=c_cached,
                           write_c_cache=str(do_c_cache).lower(),
                           snapshot_at=app.config[SNAPTIME_KEY],
                           qparams=_get_query_params())


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
    parser.add_argument('--ccache', default=None, help='client data cache')
    now = datetime.datetime.now().isoformat().replace(":", "-")
    now = now[0:19]
    parser.add_argument('--tag', default=now, help="output tag")
    parser.add_argument('--store',
                        default="/var/cache/survey/",
                        help="data store")
    parser.add_argument('--config',
                        default="/etc/survey/",
                        help="survey config files")
    parser.add_argument('--threaded',
                        default=True,
                        type=bool,
                        help="enable backend threading")
    args = parser.parse_args()
    app.config[QUESTION_KEY] = []
    app.config[ARTIFACT_KEY] = args.store
    app.config[TAG_KEY] = _clean(args.tag)
    app.config[METHOD_KEY] = globals()[OUT_METHOD + args.output]
    app.config[SNAPTIME_KEY] = args.snapshot
    app.config[CCACHE_KEY] = args.ccache
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
    app.run(host=args.host, port=args.port, threaded=args.threaded)
    exit(0)


if __name__ == "__main__":
    main()
