import json
import logging
import sys
from argparse import ArgumentParser
from enum import Enum
from typing import Dict, Set

parser = ArgumentParser(description="The util which parses autobahn testsuite results")
parser.add_argument(
    "--filepath",
    required=True,
    help="the file path which consists testsuite results",
)
parser.add_argument(
    "--ignore-non-strict",
    action="store_true",
    help="the flag which means that need to ignore non strict status as an error",
)

logger = logging.getLogger(__name__)


class BehaviorEnum(str, Enum):
    OK = "OK"
    NON_STRICT = "NON-STRICT"
    INFO = "INFORMATIONAL"
    UNIMPLEMENTED = "UNIMPLEMENTED"
    FAIL = "FAIL"

    @classmethod
    def choices(cls) -> Set[str]:
        return {str(enum) for enum in cls}


def main() -> None:
    args = parser.parse_args()

    try:
        with open(args.filepath, mode="r") as file:
            res = json.load(file)  # type: Dict[str, dict]
    except FileNotFoundError:
        logger.error(f"file {args.filepath!r} is not found")
        sys.exit(1)

    nested_keys = iter(res.keys())
    nested_key = next(nested_keys, None)

    if nested_key is None:
        logger.error("file content doesn't have json at least one key")
        sys.exit(1)

    possible_statuses = {BehaviorEnum.OK, BehaviorEnum.INFO}
    if args.ignore_non_strict:
        possible_statuses.add(BehaviorEnum.NON_STRICT)

    has_errors = False
    for case_num, case_res in res[nested_key].items():
        case_status, case_close_status = case_res["behavior"], case_res["behaviorClose"]
        if case_status not in possible_statuses:
            has_errors = True
            logger.error(f"Case num {case_num!r} finished with {case_status!r} status")
        if case_close_status not in possible_statuses:
            has_errors = True
            logger.error(f"Case num {case_num!r} finished with {case_close_status!r} close status")

    if has_errors:
        sys.exit(1)


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        logger.exception(exc)
        sys.exit(1)
