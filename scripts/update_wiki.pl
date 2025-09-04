#!/usr/bin/env perl

=head1 NAME

update_wiki.pl - Add or update content in a personal wiki

=head1 SYNOPSIS

  update_wiki.pl -u <WIKI_BASE_URL> -t <TITLE> [-f <FILE>]

  # Read from standard input
  echo "New content here" | update_wiki.pl -u http://localhost:8083 -t "My Page"
  
  # Read from file  
  update_wiki.pl -u http://localhost:8083 -t "My Page" -f content.md
  
  # Get help
  update_wiki.pl -h

=head1 DESCRIPTION

This script adds content to a personal wiki page via REST API. If the page already 
exists, it appends the new content to the end with a timestamp separator. If the 
page doesn't exist, it creates a new page with the provided content.

The script automatically handles:
- Creating new pages when they don't exist
- Appending to existing pages with timestamp separators
- Reading content from files or standard input
- Proper API authentication and error handling

=head1 OPTIONS

=over 4

=item B<-u, --url> WIKI_BASE_URL

The base URL of your wiki server (e.g., http://localhost:8083)

=item B<-t, --title> TITLE  

The title of the wiki page to create or update

=item B<-f, --file> FILE

Optional file to read content from. If not specified, reads from standard input.

=item B<-h, --help>

Show this help message and exit

=back

=head1 EXAMPLES

  # Add content from stdin
  echo "# New Section\n\nSome content" | update_wiki.pl -u http://localhost:8083 -t "Notes"
  
  # Add content from file
  update_wiki.pl -u http://localhost:8083 -t "Daily Log" -f today.md
  
  # Pipe command output to wiki
  ps aux | update_wiki.pl -u http://localhost:8083 -t "Server Status"

=head1 EXIT STATUS

Returns 0 on success, non-zero on error.

=head1 AUTHOR

Generated for Personal Wiki System

=head1 SEE ALSO

curl(1), Personal Wiki API Documentation

=cut

use strict;
use warnings;
use Getopt::Long;
use Pod::Usage;
use LWP::UserAgent;
use JSON;
use POSIX qw(strftime);
use URI::Escape;

# Global variables for options
my $wiki_url;
my $title;
my $file;
my $help;

# Parse command line options
GetOptions(
    'url|u=s'    => \$wiki_url,
    'title|t=s'  => \$title,
    'file|f=s'   => \$file,
    'help|h'     => \$help,
) or pod2usage(2);

# Show help if requested
pod2usage(1) if $help;

# Validate required options
unless ($wiki_url && $title) {
    print STDERR "Error: Both -u (URL) and -t (title) are required\n";
    pod2usage(2);
}

# Remove trailing slash from URL
$wiki_url =~ s{/$}{};

# Read content from file or stdin
my $content;
if ($file) {
    unless (-f $file) {
        die "Error: File '$file' not found\n";
    }
    open my $fh, '<', $file or die "Error: Cannot open file '$file': $!\n";
    $content = do { local $/; <$fh> };
    close $fh;
} else {
    # Read from standard input
    $content = do { local $/; <STDIN> };
}

# Trim content
$content =~ s/^\s+|\s+$//g;

unless ($content) {
    die "Error: No content provided\n";
}

# Initialize HTTP client
my $ua = LWP::UserAgent->new;
$ua->timeout(30);
$ua->agent("update_wiki.pl/1.0");

# Check if page already exists
my $encoded_title = uri_escape($title);
my $get_url = "$wiki_url/api/pages/$encoded_title";

print "Checking if page '$title' exists...\n";
my $get_response = $ua->get($get_url);

my $existing_page;
my $is_update = 0;

if ($get_response->is_success) {
    # Page exists - we'll update it
    $existing_page = decode_json($get_response->decoded_content);
    $is_update = 1;
    print "Page exists. Will append new content.\n";
} elsif ($get_response->code == 404) {
    # Page doesn't exist - we'll create it
    print "Page doesn't exist. Will create new page.\n";
} else {
    die "Error checking page existence: " . $get_response->status_line . "\n";
}

# Prepare the content
my $final_content;
my $timestamp = strftime("%Y-%m-%d %H:%M:%S", localtime);

if ($is_update) {
    # Append to existing content
    $final_content = $existing_page->{content} . "\n\n---\n\n";
    $final_content .= "## Update - $timestamp\n\n";
    $final_content .= $content;
} else {
    # New page content
    $final_content = $content;
}

# Prepare JSON payload
my $json_data = {
    title   => $title,
    content => $final_content
};

my $json = encode_json($json_data);

# Send request (POST for new, PUT for update)
my $response;
if ($is_update) {
    # Update existing page
    my $put_url = "$wiki_url/api/pages/$encoded_title";
    print "Updating existing page...\n";
    $response = $ua->put(
        $put_url,
        'Content-Type' => 'application/json',
        Content => $json
    );
} else {
    # Create new page
    my $post_url = "$wiki_url/api/pages";
    print "Creating new page...\n";
    $response = $ua->post(
        $post_url,
        'Content-Type' => 'application/json',
        Content => $json
    );
}

# Check response
if ($response->is_success) {
    my $result = decode_json($response->decoded_content);
    if ($is_update) {
        print "Successfully updated page '$title'\n";
    } else {
        print "Successfully created page '$title' (ID: $result->{id})\n";
    }
    print "Page URL: $wiki_url/#\n";
    exit 0;
} else {
    my $error_msg = $response->decoded_content || $response->status_line;
    print STDERR "Error: Failed to " . ($is_update ? "update" : "create") . " page\n";
    print STDERR "HTTP Status: " . $response->status_line . "\n";
    
    # Try to parse JSON error message
    eval {
        my $error_data = decode_json($error_msg);
        if ($error_data->{error}) {
            print STDERR "Server Error: $error_data->{error}\n";
        }
    };
    
    if ($@) {
        print STDERR "Response: $error_msg\n";
    }
    
    exit 1;
}
