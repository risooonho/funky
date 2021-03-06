package funky

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/faiface/funky/compile"
	"github.com/faiface/funky/expr"
	"github.com/faiface/funky/parse"
	"github.com/faiface/funky/runtime"

	cxr "github.com/faiface/crux/runtime"
)

func Run(main string) (value *runtime.Value, cleanup func()) {
	noStdlib := flag.Bool("nostd", false, "do not automatically include files from $FUNKY")
	stats := flag.Bool("stats", false, "print stats after running program")
	typesSandbox := flag.Bool("types", false, "start types sandbox instead of running the program")
	listDefinitions := flag.Bool("list", false, "list all the definitions instead of running the program")
	dump := flag.String("dump", "", "specify a file to dump the compiled code into")
	flag.Parse()

	compilationStart := time.Now()

	var definitions []parse.Definition

	// files from the standard library
	if funkyPath, ok := os.LookupEnv("FUNKY"); !*noStdlib && ok {
		err := filepath.Walk(funkyPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			b, err := ioutil.ReadFile(path)
			handleErrs(err)
			tokens, err := parse.Tokenize(path, string(b))
			handleErrs(err)
			defs, err := parse.Definitions(tokens)
			handleErrs(err)
			definitions = append(definitions, defs...)
			return nil
		})
		handleErrs(err)
	}

	// files included on the command line
	for _, path := range flag.Args() {
		b, err := ioutil.ReadFile(path)
		handleErrs(err)
		tokens, err := parse.Tokenize(path, string(b))
		handleErrs(err)
		defs, err := parse.Definitions(tokens)
		handleErrs(err)
		definitions = append(definitions, defs...)
	}

	if *listDefinitions {
		for _, def := range definitions {
			switch value := def.Value.(type) {
			case expr.Expr:
				fmt.Printf("%s\n", def.Name)
				fmt.Printf("  %s\n", value.TypeInfo())
				fmt.Printf("  %s\n", value.SourceInfo())
			}
		}
		os.Exit(0)
	}

	env := new(compile.Env)
	for _, def := range definitions {
		err := env.Add(def)
		handleErrs(err)
	}

	errs := env.Validate()
	handleErrs(errs...)

	if *typesSandbox {
		runTypesSandbox(env)
		os.Exit(0)
	}

	errs = env.TypeInfer()
	handleErrs(errs...)
	globalIndices, globalValues, codeIndices, codes := env.Compile(main)

	if len(globalIndices[main]) == 0 {
		handleErrs(fmt.Errorf("no %s function", main))
	}
	if len(globalIndices[main]) > 1 {
		handleErrs(fmt.Errorf("multiple %s functions", main))
	}

	if *dump != "" {
		df, err := os.Create(*dump)
		handleErrs(err)
		for name := range globalIndices {
			for i := range globalIndices[name] {
				fmt.Fprintf(df, "# %v\n", env.SourceInfo(name, i))
				fmt.Fprintf(df, "# %v\n", env.TypeInfo(name, i))
				fmt.Fprintf(df, "FUNC %s/%d\n", name, i)
				dumpCodes(df, globalIndices, &codes[codeIndices[name][i]])
				fmt.Fprintln(df)
			}
		}
		handleErrs(df.Close())
	}

	program := &runtime.Value{Globals: globalValues, Value: globalValues[globalIndices[main][0]]}

	runningStart := time.Now()

	return program, func() {
		if *stats {
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "STATS\n")
			fmt.Fprintf(os.Stderr, "reductions:       %d\n", cxr.Reductions)
			fmt.Fprintf(os.Stderr, "compilation time: %v\n", runningStart.Sub(compilationStart))
			fmt.Fprintf(os.Stderr, "running time:     %v\n", time.Since(runningStart))
		}
	}
}

func handleErrs(errs ...error) {
	bad := false
	for _, err := range errs {
		if err != nil {
			bad = true
			fmt.Fprintln(os.Stderr, err)
		}
	}
	if bad {
		os.Exit(1)
	}
}
