package pathrouter

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	ErrNotFoundHandler = errors.New("not found handler")
)

type HandlerFunc func(ctx context.Context, vars map[string]string, userId int64, update tgbotapi.Update) error

type route struct {
	pattern    *regexp.Regexp
	paramNames []string
	handler    HandlerFunc
}

type Router struct {
	routes []route
}

func NewRouter() *Router {
	return &Router{}
}

func (r *Router) HandleFunc(path string, handler HandlerFunc) {
	if path == "" {
		panic(errors.New("path cannot be empty"))
	}
	if path[0] != '/' {
		panic(errors.New("path must start with '/'"))
	}
	if path != "/" && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	regexPattern := "^"
	paramNames := []string{}
	segments := strings.Split(path, "/")

	for i, segment := range segments {
		if i > 0 {
			regexPattern += "/"
		}

		if segment == "" {
			continue
		}

		// 检查是否是参数段
		if segment[0] == '{' && segment[len(segment)-1] == '}' {
			// 提取参数内容
			paramContent := segment[1 : len(segment)-1]

			// 检查是否包含正则表达式
			colonIndex := strings.Index(paramContent, ":")

			if colonIndex != -1 {
				// 带正则表达式的参数
				paramName := paramContent[:colonIndex]
				regexStr := paramContent[colonIndex+1:]

				if paramName == "" {
					panic(errors.New("parameter name cannot be empty"))
				}

				if regexStr == "" {
					panic(errors.New("regex pattern cannot be empty"))
				}

				paramNames = append(paramNames, paramName)
				regexPattern += "(" + regexStr + ")"
			} else {
				// 简单参数
				if paramContent == "" {
					panic(errors.New("parameter name cannot be empty"))
				}

				paramNames = append(paramNames, paramContent)
				regexPattern += "([^/]+)"
			}
		} else {
			// 静态段
			regexPattern += regexp.QuoteMeta(segment)
		}
	}

	regexPattern += "$"
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		panic(fmt.Errorf("invalid route pattern: %v", err))
	}

	r.routes = append(r.routes, route{
		pattern:    regex,
		paramNames: paramNames,
		handler:    handler,
	})

}

func (r *Router) Match(path string) (HandlerFunc, map[string]string) {
	if path == "" {
		path = "/"
	}
	if path[0] != '/' {
		path = "/" + path
	}
	if path != "/" && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	params := make(map[string]string)
	for _, route := range r.routes {
		matches := route.pattern.FindStringSubmatch(path)
		if len(matches) > 0 {
			for i, name := range route.paramNames {
				if i+1 < len(matches) {
					params[name] = matches[i+1]
				}
			}
			return route.handler, params
		}
	}

	return nil, params
}

func (r *Router) Execute(ctx context.Context, path string, userId int64, update tgbotapi.Update) error {
	handler, vars := r.Match(path)
	if handler != nil {
		return handler(ctx, vars, userId, update)
	}
	return ErrNotFoundHandler
}
