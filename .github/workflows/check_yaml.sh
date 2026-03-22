#!/bin/bash

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo ""
echo -e "${CYAN}╔═══════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║     🔍 YAML Files Validation 🔍           ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════╝${NC}"
echo ""

total=0
passed=0
failed=0

for f in *.yml *.yaml; do
    if [ -f "$f" ]; then
        total=$((total + 1))
        echo -ne "  📄 ${BLUE}$f${NC} ... "

        if yamllint -d relaxed "$f" >/dev/null 2>&1; then
            echo -e "${GREEN}✅ PASS${NC}"
            passed=$((passed + 1))
        else
            echo -e "${RED}❌ FAIL${NC}"
            echo -e "    ${YELLOW}Errors:${NC}"
            yamllint -d relaxed "$f" 2>&1 | sed 's/^/      /'
            failed=$((failed + 1))
        fi
    fi
done

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  📊 Total: ${YELLOW}$total${NC}  |  ${GREEN}✅ Passed: $passed${NC}  |  ${RED}❌ Failed: $failed${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

if [ $failed -eq 0 ]; then
    echo -e "${GREEN}🎉 All YAML files are valid!${NC}"
else
    echo -e "${RED}⚠️  Some files have errors, please fix them!${NC}"
fi

echo ""
