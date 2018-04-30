package main

import (
	"testing"

	"bytes"

	. "github.com/bborbe/assert"
)

func TestEmptyBufferCreateNoLog(t *testing.T) {

	l := make(chan commit, 10)
	buffer := &bytes.Buffer{}

	if err := AssertThat(consumeCommit(l, buffer, ""), NilValue()); err != nil {
		t.Fatal(err)
	}

	if err := AssertThat(len(l), Is(0)); err != nil {
		t.Fatal(err)
	}
}

func TestReadCommit(t *testing.T) {

	l := make(chan commit, 10)
	buffer := &bytes.Buffer{}
	buffer.WriteString(`
commit abc
Author: Benjamin Borbe <bborbe@rocketnews.de>
Date:   Wed Feb 7 14:59:26 2018 +0000

    My Commit
`)

	if err := AssertThat(consumeCommit(l, buffer, ""), NilValue()); err != nil {
		t.Fatal(err)
	}

	if err := AssertThat(len(l), Is(1)); err != nil {
		t.Fatal(err)
	}

	log := <-l
	if err := AssertThat(log.Author, Is("Benjamin Borbe <bborbe@rocketnews.de>")); err != nil {
		t.Fatal(err)
	}
	if err := AssertThat(log.Message, Is("My Commit")); err != nil {
		t.Fatal(err)
	}
	if err := AssertThat(log.Date.Unix(), Is(int64(1518015566))); err != nil {
		t.Fatal(err)
	}

}

func TestTwoReadCommits(t *testing.T) {

	l := make(chan commit, 10)
	buffer := &bytes.Buffer{}
	buffer.WriteString(`
commit abc
Author: Benjamin Borbe <bborbe@rocketnews.de>
Date:   Wed Feb 7 14:59:26 2018 +0000

    My Commit1

commit edf
Author: Benjamin Borbe <bborbe@rocketnews.de>
Date:   Wed Feb 7 14:59:26 2018 +0000

    My Commit2
`)

	if err := AssertThat(consumeCommit(l, buffer, ""), NilValue()); err != nil {
		t.Fatal(err)
	}

	if err := AssertThat(len(l), Is(2)); err != nil {
		t.Fatal(err)
	}
}
