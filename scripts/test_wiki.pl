#!/usr/bin/env perl

use strict;
use warnings;
use Getopt::Long;
use LWP::UserAgent;
use JSON;
use Test::More;
use Time::HiRes qw(time);
use URI::Escape;

# --- Configuration ---
my $wiki_url;
my $help;

GetOptions(
    'url|u=s' => \$wiki_url,
    'help|h'  => \$help,
) or die "Error in command line arguments\n";

if ($help || !$wiki_url) {
    print "Usage: $0 -u <WIKI_BASE_URL>\n";
    exit 0;
}

$wiki_url =~ s{/\z}{}; # Remove trailing slash

# --- Globals ---
my $ua = LWP::UserAgent->new(timeout => 10);
$ua->agent("test_wiki.pl/1.0");

my $json_codec = JSON->new->utf8;

# Generate a unique title for the test page to avoid collisions
my $test_page_title = "Test Page " . int(time());
my $encoded_title = uri_escape($test_page_title);

# --- Plan Tests ---
plan tests => 8;

# --- Test Cases ---

# 1. Check if the API is reachable
subtest 'API Reachability' => sub {
    plan tests => 2;
    my $res = $ua->get("$wiki_url/api/pages");
    ok($res->is_success, "Successfully connected to the API");
    is($res->header('Content-Type'), 'application/json', "API returns JSON");
};

# 2. Create a new page
my $page_content = "This is the initial content.";
subtest 'Create Page' => sub {
    plan tests => 2;
    my $payload = $json_codec->encode({
        title   => $test_page_title,
        content => $page_content,
    });
    my $res = $ua->post("$wiki_url/api/pages", 'Content-Type' => 'application/json', Content => $payload);
    ok($res->is_success, "POST request to create page was successful");
    is($res->code, 201, "HTTP status is 201 Created");
};

# 3. Verify page creation
subtest 'Verify Page Creation' => sub {
    plan tests => 2;
    my $res = $ua->get("$wiki_url/api/pages/$encoded_title");
    ok($res->is_success, "GET request for new page was successful");
    my $page = $json_codec->decode($res->decoded_content);
    is($page->{content}, $page_content, "Page content matches initial content");
};

# 4. Update the page
my $updated_content = "This is the updated content.";
subtest 'Update Page' => sub {
    plan tests => 1;
    my $payload = $json_codec->encode({
        title   => $test_page_title,
        content => $updated_content,
    });
    my $res = $ua->put("$wiki_url/api/pages/$encoded_title", 'Content-Type' => 'application/json', Content => $payload);
    ok($res->is_success, "PUT request to update page was successful");
};

# 5. Verify page update
subtest 'Verify Page Update' => sub {
    plan tests => 2;
    my $res = $ua->get("$wiki_url/api/pages/$encoded_title");
    ok($res->is_success, "GET request for updated page was successful");
    my $page = $json_codec->decode($res->decoded_content);
    is($page->{content}, $updated_content, "Page content was updated correctly");
};

# 6. Delete the page
subtest 'Delete Page' => sub {
    plan tests => 2;
    my $res = $ua->delete("$wiki_url/api/pages/$encoded_title");
    ok($res->is_success, "DELETE request was successful");
    is($res->code, 204, "HTTP status is 204 No Content");
};

# 7. Verify page deletion
subtest 'Verify Page Deletion' => sub {
    plan tests => 1;
    my $res = $ua->get("$wiki_url/api/pages/$encoded_title");
    is($res->code, 404, "Page is no longer found (404)");
};

# 8. Check search functionality
subtest 'Search Functionality' => sub {
    plan tests => 3;
    
    # Create a page to search for
    my $search_title = "Searchable Test Page " . int(time());
    my $search_content = "unique_search_term_xyz";
    my $payload = $json_codec->encode({ title => $search_title, content => $search_content });
    my $res = $ua->post("$wiki_url/api/pages", 'Content-Type' => 'application/json', Content => $payload);
    BAIL_OUT("Failed to create page for searching") unless $res->is_success;

    # Search for the page
    my $search_res = $ua->get("$wiki_url/api/pages/search?q=$search_content");
    ok($search_res->is_success, "Search request was successful");
    
    my $results = $json_codec->decode($search_res->decoded_content);
    is(scalar(@$results), 1, "Search returned one result");
    is($results->[0]->{title}, $search_title, "Search returned the correct page");

    # Cleanup
    $ua->delete("$wiki_url/api/pages/" . uri_escape($search_title));
};

done_testing();
