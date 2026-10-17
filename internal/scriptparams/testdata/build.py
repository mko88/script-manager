#!/usr/bin/env python3
"""Build tool."""
import argparse

parser = argparse.ArgumentParser(description="Builds the "
                                 "project.")
parser.add_argument("target", help="what to build")
parser.add_argument("extra", nargs="*", help="extra targets")
parser.add_argument("-v", "--verbose", action="store_true", help="talk more")
parser.add_argument("--no-cache", dest="cache", action="store_false",
                    help="skip the cache")
parser.add_argument(
    "-m", "--mode",
    choices=["debug", "release"],
    default="debug",
    help="build mode (default: %(default)s)",
)
parser.add_argument("--arch", nargs="+", choices=("x64", "arm64"), help='architectures, "quoted"')
parser.add_argument("--define", "-D", action="append", help="a define")
parser.add_argument("--jobs", type=int, default=4, required=True, help=r"parallel jobs\n")
parser.add_argument("--secret", help=argparse.SUPPRESS)
# parser.add_argument("--commented", help="not real")
parser.add_argument("--retries", choices=RETRY_CHOICES, help="tries")
args = parser.parse_args()
